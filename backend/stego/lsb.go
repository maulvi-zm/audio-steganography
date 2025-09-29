// Package stego to implement LSB
package stego

import (
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"math/rand"
	"steganography-backend/crypto"
	"steganography-backend/models"
)

type LSBSteganography struct {
	config *models.StegoConfig
	rng    *rand.Rand
}

func NewLSBSteganography(config *models.StegoConfig) *LSBSteganography {
	// Create deterministic random number generator from key
	seed := generateSeed(config.Key)
	rng := rand.New(rand.NewSource(seed))

	return &LSBSteganography{
		config: config,
		rng:    rng,
	}
}

func generateSeed(key string) int64 {
	hash := md5.Sum([]byte(key))
	return int64(binary.BigEndian.Uint64(hash[:8]))
}

func (lsb *LSBSteganography) CalculateCapacity(pcmData []byte) int {
	// We need space for: filename length (4 bytes) + filename + data length (4 bytes) + data
	bitsPerByte := lsb.config.LSBBits
	totalBits := len(pcmData) * bitsPerByte
	totalBytes := totalBits / 8

	// Reserve space for metadata (filename length + data length)
	metadataBytes := 8 // 4 bytes for filename length + 4 bytes for data length
	return totalBytes - metadataBytes
}

func (lsb *LSBSteganography) Embed(pcmData []byte, secretData []byte) ([]byte, error) {
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
	capacity := lsb.CalculateCapacity(pcmData)
	if len(payload) > capacity {
		return nil, fmt.Errorf("secret data too large: %d bytes, capacity: %d bytes", len(payload), capacity)
	}

	// Create copy of PCM data for modification
	stegoPCM := make([]byte, len(pcmData))
	copy(stegoPCM, pcmData)

	// Calculate how many audio samples we need based on LSB bits per sample
	totalPayloadBits := len(payload) * 8
	samplesNeeded := totalPayloadBits / lsb.config.LSBBits
	if totalPayloadBits%lsb.config.LSBBits != 0 {
		samplesNeeded++
	}

	positions := lsb.generatePositions(len(pcmData), samplesNeeded)

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

		// Clear the LSB bits and set new ones
		stegoPCM[pos] = (stegoPCM[pos] & ^mask) | (bitsToEmbed & mask)
	}

	return stegoPCM, nil
}

func (lsb *LSBSteganography) Extract(stegoPCM []byte) ([]byte, string, error) {

	// First extract enough samples to get the filename length (4 bytes = 32 bits)
	filenameMetadataBits := 32
	samplesNeeded := filenameMetadataBits / lsb.config.LSBBits
	if filenameMetadataBits%lsb.config.LSBBits != 0 {
		samplesNeeded++
	}

	positions := lsb.generatePositions(len(stegoPCM), samplesNeeded)
	if len(positions) < samplesNeeded {
		return nil, "", fmt.Errorf("insufficient data for metadata extraction")
	}

	// Extract bits to reconstruct filename length
	extractedBits := make([]byte, 0)
	mask := byte((1 << lsb.config.LSBBits) - 1)

	for i := 0; i < samplesNeeded; i++ {
		lsbValue := stegoPCM[positions[i]] & mask
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
	samplesNeeded = metadataBits / lsb.config.LSBBits
	if metadataBits%lsb.config.LSBBits != 0 {
		samplesNeeded++
	}

	positions = lsb.generatePositions(len(stegoPCM), samplesNeeded)
	if len(positions) < samplesNeeded {
		return nil, "", fmt.Errorf("insufficient positions for metadata extraction")
	}

	// Extract all metadata bits
	extractedBits = make([]byte, 0)
	for i := 0; i < samplesNeeded; i++ {
		lsbValue := stegoPCM[positions[i]] & mask
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
	samplesNeeded = totalPayloadBits / lsb.config.LSBBits
	if totalPayloadBits%lsb.config.LSBBits != 0 {
		samplesNeeded++
	}

	positions = lsb.generatePositions(len(stegoPCM), samplesNeeded)
	if len(positions) < samplesNeeded {
		return nil, "", fmt.Errorf("insufficient positions for complete payload extraction")
	}

	// Extract all payload bits
	extractedBits = make([]byte, 0)
	for i := 0; i < samplesNeeded; i++ {
		lsbValue := stegoPCM[positions[i]] & mask
		// Unpack bits from this LSB value
		for j := 0; j < lsb.config.LSBBits && len(extractedBits) < totalPayloadBits; j++ {
			extractedBits = append(extractedBits, (lsbValue>>j)&1)
		}
	}

	extractedBytes = bitsToBytes(extractedBits[:totalPayloadBits])

	// Decrypt the entire payload if encryption was used
	if lsb.config.UseEncryption {
		cipher := crypto.NewExtendedVigenere(lsb.config.Key)
		extractedBytes = cipher.Decrypt(extractedBytes)
	}

	// Re-parse decrypted metadata to get correct filename and data length
	filenameLen = binary.BigEndian.Uint32(extractedBytes[0:4])
	if filenameLen > 255 {
		return nil, "", fmt.Errorf("invalid decrypted filename length: %d", filenameLen)
	}

	filename = string(extractedBytes[4 : 4+filenameLen])
	dataLen = binary.BigEndian.Uint32(extractedBytes[4+filenameLen : 4+filenameLen+4])
	if dataLen > 10*1024*1024 { // 10MB sanity check
		return nil, "", fmt.Errorf("invalid decrypted data length: %d", dataLen)
	}

	dataStart := 4 + filenameLen + 4
	if int(dataStart+dataLen) > len(extractedBytes) {
		return nil, "", fmt.Errorf("insufficient extracted data: expected %d bytes, got %d", dataLen, len(extractedBytes)-int(dataStart))
	}

	secretData := extractedBytes[dataStart : dataStart+dataLen]

	return secretData, filename, nil
}

func (lsb *LSBSteganography) generatePositions(audioLen, bitsNeeded int) []int {
	positions := make([]int, 0)

	if lsb.config.UseRandomStart {
		used := make(map[int]bool)
		for len(positions) < bitsNeeded && len(positions) < audioLen {
			pos := lsb.rng.Intn(audioLen)
			if !used[pos] {
				positions = append(positions, pos)
				used[pos] = true
			}
		}
	} else {
		for i := 0; i < bitsNeeded && i < audioLen; i++ {
			positions = append(positions, i)
		}
	}

	return positions
}

func bytesToBits(data []byte) []byte {
	bits := make([]byte, 0)
	for _, b := range data {
		for i := 7; i >= 0; i-- {
			bits = append(bits, (b>>i)&1)
		}
	}
	return bits
}

func bitsToBytes(bits []byte) []byte {
	bytes := make([]byte, 0)
	for i := 0; i < len(bits); i += 8 {
		if i+8 > len(bits) {
			break
		}
		var b byte
		for j := range 8 {
			b = (b << 1) | (bits[i+j] & 1)
		}
		bytes = append(bytes, b)
	}
	return bytes
}
