package stego

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"steganography-backend/crypto"
	"steganography-backend/models"
	"steganography-backend/mp3parser"
)

type MP3LSBSteganography struct {
	config *models.StegoConfig
	rng    *rand.Rand
}

func NewMP3LSBSteganography(config *models.StegoConfig) *MP3LSBSteganography {
	// Create deterministic random number generator from key
	seed := generateSeed(config.Key)
	rng := rand.New(rand.NewSource(seed))

	return &MP3LSBSteganography{
		config: config,
		rng:    rng,
	}
}

func (lsb *MP3LSBSteganography) CalculateCapacity(mp3Data []byte) (int, error) {
	mp3File, err := mp3parser.ParseMP3File(mp3Data)
	if err != nil {
		return 0, fmt.Errorf("failed to parse MP3: %v", err)
	}

	totalBytes := 0
	for _, frame := range mp3File.Frames {
		// Use frame data for embedding (skip frame headers and some reserved bytes)
		usableBytes := len(frame.Data) - 10 // Reserve some bytes for frame integrity
		if usableBytes > 0 {
			totalBytes += usableBytes
		}
	}

	bitsPerByte := lsb.config.LSBBits
	totalBits := totalBytes * bitsPerByte
	capacity := totalBits / 8

	// Reserve space for metadata (filename length + data length)
	metadataBytes := 8 // 4 bytes for filename length + 4 bytes for data length
	return capacity - metadataBytes, nil
}

func (lsb *MP3LSBSteganography) EmbedInMP3(mp3Data []byte, secretData []byte) ([]byte, error) {
	if lsb.config.UseEncryption {
		cipher := crypto.NewExtendedVigenere(lsb.config.Key)
		secretData = cipher.Encrypt(secretData)
	}

	// Parse MP3 file
	mp3File, err := mp3parser.ParseMP3File(mp3Data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse MP3: %v", err)
	}

	// Prepare payload: filename length + filename + data length + data
	filename := []byte(lsb.config.SecretFilename)
	payload := make([]byte, 0)

	// Add filename length (4 bytes)
	filenameLen := make([]byte, 4)
	binary.BigEndian.PutUint32(filenameLen, uint32(len(filename)))
	payload = append(payload, filenameLen...)

	// Add filename
	payload = append(payload, filename...)

	// Add data length (4 bytes)
	dataLen := make([]byte, 4)
	binary.BigEndian.PutUint32(dataLen, uint32(len(secretData)))
	payload = append(payload, dataLen...)

	// Add secret data
	payload = append(payload, secretData...)

	// Check capacity
	capacity, err := lsb.CalculateCapacity(mp3Data)
	if err != nil {
		return nil, err
	}
	if len(payload) > capacity {
		return nil, fmt.Errorf("secret data too large: %d bytes, capacity: %d bytes", len(payload), capacity)
	}

	// Collect all usable bytes from frame data
	allFrameData := make([]byte, 0)

	for _, frame := range mp3File.Frames {
		usableBytes := len(frame.Data) - 10 // Reserve some bytes for frame integrity
		if usableBytes > 0 {
			allFrameData = append(allFrameData, frame.Data[:usableBytes]...)
		}
	}

	// Calculate how many bytes we need based on LSB bits per byte
	totalPayloadBits := len(payload) * 8
	bytesNeeded := totalPayloadBits / lsb.config.LSBBits
	if totalPayloadBits%lsb.config.LSBBits != 0 {
		bytesNeeded++
	}

	positions := lsb.generatePositions(len(allFrameData), bytesNeeded)

	// Convert payload to bits for multi-bit embedding
	payloadBits := bytesToBits(payload)

	// Embed bits using LSBBits per position
	mask := byte((1 << lsb.config.LSBBits) - 1)
	bitIndex := 0

	for _, pos := range positions {
		if bitIndex >= len(payloadBits) {
			break
		}

		// Pack multiple bits into LSB positions
		var bitsToEmbed byte = 0
		for j := 0; j < lsb.config.LSBBits && bitIndex < len(payloadBits); j++ {
			bitsToEmbed |= (payloadBits[bitIndex] << j)
			bitIndex++
		}

		// Modify the byte in allFrameData
		allFrameData[pos] = (allFrameData[pos] & ^mask) | (bitsToEmbed & mask)
	}

	// Put modified data back into frames
	dataIndex := 0
	for _, frame := range mp3File.Frames {
		usableBytes := len(frame.Data) - 10
		if usableBytes > 0 {
			copy(frame.Data[:usableBytes], allFrameData[dataIndex:dataIndex+usableBytes])
			dataIndex += usableBytes
		}
	}

	// Reconstruct MP3 file
	return mp3parser.WriteMP3File(mp3File)
}

func (lsb *MP3LSBSteganography) ExtractFromMP3(mp3Data []byte) ([]byte, string, error) {
	// Parse MP3 file
	mp3File, err := mp3parser.ParseMP3File(mp3Data)
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse MP3: %v", err)
	}

	// Collect all usable bytes from frame data
	allFrameData := make([]byte, 0)
	for _, frame := range mp3File.Frames {
		usableBytes := len(frame.Data) - 10 // Same reservation as in embedding
		if usableBytes > 0 {
			allFrameData = append(allFrameData, frame.Data[:usableBytes]...)
		}
	}

	if len(allFrameData) == 0 {
		return nil, "", fmt.Errorf("no usable frame data found")
	}

	// First extract enough bytes to get the filename length (4 bytes = 32 bits)
	filenameMetadataBits := 32
	bytesNeeded := filenameMetadataBits / lsb.config.LSBBits
	if filenameMetadataBits%lsb.config.LSBBits != 0 {
		bytesNeeded++
	}

	positions := lsb.generatePositions(len(allFrameData), bytesNeeded)
	if len(positions) < bytesNeeded {
		return nil, "", fmt.Errorf("insufficient data for metadata extraction")
	}

	// Extract bits to reconstruct filename length
	extractedBits := make([]byte, 0)
	mask := byte((1 << lsb.config.LSBBits) - 1)

	for i := 0; i < bytesNeeded; i++ {
		lsbValue := allFrameData[positions[i]] & mask
		// Unpack bits from this LSB value
		for j := 0; j < lsb.config.LSBBits && len(extractedBits) < filenameMetadataBits; j++ {
			extractedBits = append(extractedBits, (lsbValue>>j)&1)
		}
	}

	// Convert bits to bytes to get filename length
	filenameBytes := bitsToBytes(extractedBits[:filenameMetadataBits])
	if len(filenameBytes) < 4 {
		return nil, "", fmt.Errorf("insufficient data for filename length")
	}

	filenameLen := binary.BigEndian.Uint32(filenameBytes[0:4])
	if filenameLen > 255 {
		return nil, "", fmt.Errorf("invalid filename length: %d", filenameLen)
	}

	// Now extract enough for: filename length (4) + filename + data length (4)
	metadataBits := (8 + int(filenameLen)) * 8
	bytesNeeded = metadataBits / lsb.config.LSBBits
	if metadataBits%lsb.config.LSBBits != 0 {
		bytesNeeded++
	}

	positions = lsb.generatePositions(len(allFrameData), bytesNeeded)
	if len(positions) < bytesNeeded {
		return nil, "", fmt.Errorf("insufficient positions for metadata extraction")
	}

	// Extract all metadata bits
	extractedBits = make([]byte, 0)
	for i := 0; i < bytesNeeded; i++ {
		lsbValue := allFrameData[positions[i]] & mask
		// Unpack bits from this LSB value
		for j := 0; j < lsb.config.LSBBits && len(extractedBits) < metadataBits; j++ {
			extractedBits = append(extractedBits, (lsbValue>>j)&1)
		}
	}

	extractedBytes := bitsToBytes(extractedBits[:metadataBits])

	if len(extractedBytes) < int(8+filenameLen) {
		return nil, "", fmt.Errorf("insufficient extracted data for metadata")
	}

	// Parse filename
	filename := string(extractedBytes[4 : 4+filenameLen])

	dataLen := binary.BigEndian.Uint32(extractedBytes[4+filenameLen : 4+filenameLen+4])
	if dataLen > 10*1024*1024 { // 10MB sanity check
		return nil, "", fmt.Errorf("invalid data length: %d", dataLen)
	}

	// Now extract the complete payload: metadata + actual data
	totalPayloadBits := (8 + int(filenameLen) + int(dataLen)) * 8
	bytesNeeded = totalPayloadBits / lsb.config.LSBBits
	if totalPayloadBits%lsb.config.LSBBits != 0 {
		bytesNeeded++
	}

	positions = lsb.generatePositions(len(allFrameData), bytesNeeded)
	if len(positions) < bytesNeeded {
		return nil, "", fmt.Errorf("insufficient positions for complete payload extraction")
	}

	// Extract all payload bits
	extractedBits = make([]byte, 0)
	for i := 0; i < bytesNeeded; i++ {
		lsbValue := allFrameData[positions[i]] & mask
		// Unpack bits from this LSB value
		for j := 0; j < lsb.config.LSBBits && len(extractedBits) < totalPayloadBits; j++ {
			extractedBits = append(extractedBits, (lsbValue>>j)&1)
		}
	}

	extractedBytes = bitsToBytes(extractedBits[:totalPayloadBits])

	dataStart := 4 + filenameLen + 4
	if int(dataStart+dataLen) > len(extractedBytes) {
		return nil, "", fmt.Errorf("insufficient extracted data: expected %d bytes, got %d", dataLen, len(extractedBytes)-int(dataStart))
	}

	secretData := extractedBytes[dataStart : dataStart+dataLen]

	// Decrypt if encryption was used
	if lsb.config.UseEncryption {
		cipher := crypto.NewExtendedVigenere(lsb.config.Key)
		secretData = cipher.Decrypt(secretData)
	}

	return secretData, filename, nil
}

func (lsb *MP3LSBSteganography) generatePositions(dataLen, bytesNeeded int) []int {
	positions := make([]int, 0)

	if lsb.config.UseRandomStart {
		used := make(map[int]bool)
		for len(positions) < bytesNeeded && len(positions) < dataLen {
			pos := lsb.rng.Intn(dataLen)
			if !used[pos] {
				positions = append(positions, pos)
				used[pos] = true
			}
		}
	} else {
		for i := 0; i < bytesNeeded && i < dataLen; i++ {
			positions = append(positions, i)
		}
	}

	return positions
}
