package stego

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"steganography-backend/crypto"
	"steganography-backend/models"
	"steganography-backend/mp3parser"
)

type MP3AncillaryLSBSteganography struct {
	config *models.StegoConfig
	rng    *rand.Rand
}

func NewMP3AncillaryLSBSteganography(config *models.StegoConfig) *MP3AncillaryLSBSteganography {
	// Create deterministic random number generator from key
	seed := generateSeed(config.Key)
	rng := rand.New(rand.NewSource(seed))

	return &MP3AncillaryLSBSteganography{
		config: config,
		rng:    rng,
	}
}

func (lsb *MP3AncillaryLSBSteganography) CalculateCapacity(mp3Data []byte) (int, error) {
	mp3File, err := mp3parser.ParseMP3File(mp3Data)
	if err != nil {
		return 0, fmt.Errorf("failed to parse MP3: %v", err)
	}

	totalSafeBytes := 0
	for _, frame := range mp3File.Frames {
		regions, err := mp3parser.AnalyzeFrameData(frame.Header, frame.Data)
		if err != nil {
			continue // Skip problematic frames
		}

		safeBytes := regions.GetSafeModificationBytes()
		totalSafeBytes += len(safeBytes)
	}

	if totalSafeBytes == 0 {
		return 0, fmt.Errorf("no safe ancillary data found in MP3 frames")
	}

	bitsPerByte := lsb.config.LSBBits
	totalBits := totalSafeBytes * bitsPerByte
	capacity := totalBits / 8

	// Reserve space for metadata (filename length + data length)
	metadataBytes := 8 // 4 bytes for filename length + 4 bytes for data length
	if capacity < metadataBytes {
		return 0, fmt.Errorf("insufficient ancillary data for metadata")
	}

	return capacity - metadataBytes, nil
}

func (lsb *MP3AncillaryLSBSteganography) EmbedInMP3(mp3Data []byte, secretData []byte) ([]byte, error) {
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

	// Encrypt the entire payload if encryption is enabled
	if lsb.config.UseEncryption {
		cipher := crypto.NewExtendedVigenere(lsb.config.Key)
		payload = cipher.Encrypt(payload)
	}

	// Parse MP3 file
	mp3File, err := mp3parser.ParseMP3File(mp3Data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse MP3: %v", err)
	}

	// Check capacity
	capacity, err := lsb.CalculateCapacity(mp3Data)
	if err != nil {
		return nil, err
	}
	if len(payload) > capacity {
		return nil, fmt.Errorf("secret data too large: %d bytes, capacity: %d bytes", len(payload), capacity)
	}

	// Collect all safe bytes from all frames
	allSafeBytes := make([]byte, 0)
	frameRegions := make([]*mp3parser.MP3FrameRegions, 0)

	for _, frame := range mp3File.Frames {
		regions, err := mp3parser.AnalyzeFrameData(frame.Header, frame.Data)
		if err != nil {
			// Create empty regions for problematic frames
			regions = &mp3parser.MP3FrameRegions{}
		}

		frameRegions = append(frameRegions, regions)
		safeBytes := regions.GetSafeModificationBytes()
		allSafeBytes = append(allSafeBytes, safeBytes...)
	}

	if len(allSafeBytes) == 0 {
		return nil, fmt.Errorf("no safe ancillary data available for embedding")
	}

	// Calculate how many bytes we need based on LSB bits per byte
	totalPayloadBits := len(payload) * 8
	bytesNeeded := totalPayloadBits / lsb.config.LSBBits
	if totalPayloadBits%lsb.config.LSBBits != 0 {
		bytesNeeded++
	}

	if bytesNeeded > len(allSafeBytes) {
		return nil, fmt.Errorf("insufficient safe bytes: need %d, have %d", bytesNeeded, len(allSafeBytes))
	}

	positions := lsb.generatePositions(len(allSafeBytes), bytesNeeded)

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

		// Modify the byte in allSafeBytes
		allSafeBytes[pos] = (allSafeBytes[pos] & ^mask) | (bitsToEmbed & mask)
	}

	// Put modified safe data back into frames
	safeByteIndex := 0
	for i, frame := range mp3File.Frames {
		regions := frameRegions[i]
		originalSafeBytes := regions.GetSafeModificationBytes()

		if len(originalSafeBytes) > 0 {
			// Extract the modified safe bytes for this frame
			modifiedSafeBytes := allSafeBytes[safeByteIndex : safeByteIndex+len(originalSafeBytes)]
			safeByteIndex += len(originalSafeBytes)

			// Reconstruct frame data with modified safe bytes
			frame.Data = regions.ReconstructFrameData(modifiedSafeBytes)
		}
	}

	// Reconstruct MP3 file
	return mp3parser.WriteMP3File(mp3File)
}

func (lsb *MP3AncillaryLSBSteganography) ExtractFromMP3(mp3Data []byte) ([]byte, string, error) {
	// Parse MP3 file
	mp3File, err := mp3parser.ParseMP3File(mp3Data)
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse MP3: %v", err)
	}

	// Collect all safe bytes from all frames
	allSafeBytes := make([]byte, 0)
	for _, frame := range mp3File.Frames {
		regions, err := mp3parser.AnalyzeFrameData(frame.Header, frame.Data)
		if err != nil {
			continue // Skip problematic frames
		}

		safeBytes := regions.GetSafeModificationBytes()
		allSafeBytes = append(allSafeBytes, safeBytes...)
	}

	if len(allSafeBytes) == 0 {
		return nil, "", fmt.Errorf("no safe ancillary data found")
	}

	// CRITICAL FIX: Generate the complete position permutation
	// This creates the same random sequence that embedding used
	// We'll extract as much as possible and parse sequentially

	// Generate positions for ALL available safe bytes to get the complete permutation
	positions := lsb.generatePositions(len(allSafeBytes), len(allSafeBytes))
	if len(positions) == 0 {
		return nil, "", fmt.Errorf("no positions generated for extraction")
	}

	// Extract all available bits using the complete position sequence
	extractedBits := make([]byte, 0)
	mask := byte((1 << lsb.config.LSBBits) - 1)

	for i := 0; i < len(positions); i++ {
		lsbValue := allSafeBytes[positions[i]] & mask
		// Unpack bits from this LSB value
		for j := 0; j < lsb.config.LSBBits; j++ {
			extractedBits = append(extractedBits, (lsbValue>>j)&1)
		}
	}

	// Now parse the extracted bits sequentially
	extractedBytes := bitsToBytes(extractedBits)

	if len(extractedBytes) < 8 {
		return nil, "", fmt.Errorf("insufficient extracted data for basic metadata")
	}

	// Decrypt the entire payload if encryption was used
	if lsb.config.UseEncryption {
		cipher := crypto.NewExtendedVigenere(lsb.config.Key)
		extractedBytes = cipher.Decrypt(extractedBytes)
	}

	// Parse filename length
	filenameLen := binary.BigEndian.Uint32(extractedBytes[0:4])
	if filenameLen > 255 {
		return nil, "", fmt.Errorf("invalid filename length: %d", filenameLen)
	}

	if len(extractedBytes) < int(8+filenameLen) {
		return nil, "", fmt.Errorf("insufficient extracted data for filename")
	}

	// Parse filename
	filename := string(extractedBytes[4 : 4+filenameLen])

	// Parse data length
	dataLen := binary.BigEndian.Uint32(extractedBytes[4+filenameLen : 4+filenameLen+4])
	if dataLen > 10*1024*1024 { // 10MB sanity check
		return nil, "", fmt.Errorf("invalid data length: %d", dataLen)
	}

	dataStart := 4 + filenameLen + 4
	if int(dataStart+dataLen) > len(extractedBytes) {
		return nil, "", fmt.Errorf("insufficient extracted data: expected %d bytes, got %d", dataLen, len(extractedBytes)-int(dataStart))
	}

	secretData := extractedBytes[dataStart : dataStart+dataLen]

	return secretData, filename, nil
}

func (lsb *MP3AncillaryLSBSteganography) generatePositions(dataLen, bytesNeeded int) []int {
	positions := make([]int, 0)

	if lsb.config.UseRandomStart {
		// CRITICAL FIX: Reset RNG to ensure consistent sequence between embed/extract
		seed := generateSeed(lsb.config.Key)
		lsb.rng.Seed(seed)

		// Generate a FIXED permutation of all available positions
		used := make(map[int]bool)
		allPositions := make([]int, 0, dataLen)

		// Generate the complete random permutation of all available positions
		for len(allPositions) < dataLen {
			pos := lsb.rng.Intn(dataLen)
			if !used[pos] {
				allPositions = append(allPositions, pos)
				used[pos] = true
			}
		}

		// Return only the first bytesNeeded positions from the fixed permutation
		if bytesNeeded > len(allPositions) {
			bytesNeeded = len(allPositions)
		}
		positions = allPositions[:bytesNeeded]
	} else {
		for i := 0; i < bytesNeeded && i < dataLen; i++ {
			positions = append(positions, i)
		}
	}

	return positions
}
