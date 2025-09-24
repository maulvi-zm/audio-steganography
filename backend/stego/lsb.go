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
	if lsb.config.UseEncryption {
		cipher := crypto.NewExtendedVigenere(lsb.config.Key)
		secretData = cipher.Encrypt(secretData)
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

	// Add secret data and check the capacity
	payload = append(payload, secretData...)
	capacity := lsb.CalculateCapacity(pcmData)
	if len(payload) > capacity {
		return nil, fmt.Errorf("secret data too large: %d bytes, capacity: %d bytes", len(payload), capacity)
	}

	payloadBits := bytesToBits(payload)

	// Create copy of PCM data for modification
	stegoPCM := make([]byte, len(pcmData))
	copy(stegoPCM, pcmData)

	positions := lsb.generatePositions(len(pcmData), len(payloadBits))

	// Embed bits
	bitIndex := 0
	for _, pos := range positions {
		if bitIndex >= len(payloadBits) {
			break
		}

		// Clear the LSB bits and set new ones
		mask := byte((1 << lsb.config.LSBBits) - 1)
		stegoPCM[pos] = (stegoPCM[pos] & ^mask) | (payloadBits[bitIndex] & mask)
		bitIndex++
	}

	return stegoPCM, nil
}

func (lsb *LSBSteganography) Extract(stegoPCM []byte) ([]byte, string, error) {
	positions := lsb.generatePositions(len(stegoPCM), len(stegoPCM)*lsb.config.LSBBits)

	metadataBits := 32 // 4 bytes for filename length
	if len(positions) < metadataBits/lsb.config.LSBBits {
		return nil, "", fmt.Errorf("insufficient data for metadata extraction")
	}

	extractedBits := make([]byte, 0)
	mask := byte((1 << lsb.config.LSBBits) - 1)

	// Extract bits according to positions
	for i, pos := range positions {
		if i >= len(stegoPCM)*lsb.config.LSBBits {
			break
		}
		extractedBits = append(extractedBits, stegoPCM[pos]&mask)
	}

	extractedBytes := bitsToBytes(extractedBits)

	if len(extractedBytes) < 8 { // minimum: 4 bytes filename length + 4 bytes data length
		return nil, "", fmt.Errorf("insufficient extracted data")
	}

	// Parse filename length
	filenameLen := binary.BigEndian.Uint32(extractedBytes[0:4])
	if filenameLen > 255 || int(filenameLen) > len(extractedBytes)-8 {
		return nil, "", fmt.Errorf("invalid filename length: %d", filenameLen)
	}

	filename := string(extractedBytes[4 : 4+filenameLen])
	if len(extractedBytes) < int(4+filenameLen+4) {
		return nil, "", fmt.Errorf("insufficient data for data length")
	}

	dataLen := binary.BigEndian.Uint32(extractedBytes[4+filenameLen : 4+filenameLen+4])
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
