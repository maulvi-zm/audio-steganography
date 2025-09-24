// Package handlers is made to handle requests
package handlers

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"steganography-backend/audio"
	"steganography-backend/crypto"
	"steganography-backend/models"
	"steganography-backend/stego"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type StegoHandler struct {
	audioDecoder *audio.AudioDecoder
}

func NewStegoHandler() *StegoHandler {
	return &StegoHandler{
		audioDecoder: audio.NewAudioDecoder(),
	}
}

func (h *StegoHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"message": "Steganography API is running",
		"version": "1.0.0",
	})
}

func (h *StegoHandler) InsertMessage(c *gin.Context) {
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil { // 32MB limit
		c.JSON(http.StatusBadRequest, models.StegoResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to parse form: %v", err),
		})
		return
	}

	key := c.PostForm("key")
	useEncryption := c.PostForm("use_encryption") == "true"
	useRandomStart := c.PostForm("use_random_start") == "true"
	lsbBitsStr := c.PostForm("lsb_bits")

	if key == "" {
		c.JSON(http.StatusBadRequest, models.StegoResponse{
			Success: false,
			Message: "Key is required",
		})
		return
	}

	if err := crypto.ValidateKey(key); err != nil {
		c.JSON(http.StatusBadRequest, models.StegoResponse{
			Success: false,
			Message: fmt.Sprintf("Invalid key: %v", err),
		})
		return
	}

	lsbBits, err := strconv.Atoi(lsbBitsStr)
	if err != nil || lsbBits < 1 || lsbBits > 4 {
		c.JSON(http.StatusBadRequest, models.StegoResponse{
			Success: false,
			Message: "LSB bits must be between 1 and 4",
		})
		return
	}

	// Get uploaded files
	audioFile, audioHeader, err := c.Request.FormFile("audio_file")
	if err != nil {
		c.JSON(http.StatusBadRequest, models.StegoResponse{
			Success: false,
			Message: "Audio file is required",
		})
		return
	}
	defer audioFile.Close()

	secretFile, secretHeader, err := c.Request.FormFile("secret_file")
	if err != nil {
		c.JSON(http.StatusBadRequest, models.StegoResponse{
			Success: false,
			Message: "Secret file is required",
		})
		return
	}
	defer secretFile.Close()

	if !isValidMP3File(audioHeader.Filename) {
		c.JSON(http.StatusBadRequest, models.StegoResponse{
			Success: false,
			Message: "Invalid audio file format. Only MP3 files are supported",
		})
		return
	}

	// Read audio file
	audioData, err := io.ReadAll(audioFile)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.StegoResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to read audio file: %v", err),
		})
		return
	}

	// Decode MP3 to PCM data
	pcmData, audioMetadata, err := h.audioDecoder.DecodeMP3(audioData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.StegoResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to decode MP3 file: %v", err),
		})
		return
	}

	// Read secret file
	secretData, err := io.ReadAll(secretFile)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.StegoResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to read secret file: %v", err),
		})
		return
	}

	config := &models.StegoConfig{
		Key:            key,
		UseEncryption:  useEncryption,
		UseRandomStart: useRandomStart,
		LSBBits:        lsbBits,
		SecretFilename: secretHeader.Filename,
	}

	lsbStego := stego.NewLSBSteganography(config)
	capacity := lsbStego.CalculateCapacity(pcmData)
	if len(secretData)+len(secretHeader.Filename)+8 > capacity {
		c.JSON(http.StatusBadRequest, models.StegoResponse{
			Success: false,
			Message: fmt.Sprintf("Secret data too large. Maximum capacity: %d bytes, required: %d bytes",
				capacity, len(secretData)+len(secretHeader.Filename)+8),
		})
		return
	}

	// Embed secret data into PCM
	stegoPCM, err := lsbStego.Embed(pcmData, secretData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.StegoResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to embed secret data: %v", err),
		})
		return
	}

	// Calculate PSNR between original and stego PCM data
	psnr := audio.CalculatePSNRFloat64(bytesToFloat64(pcmData), bytesToFloat64(stegoPCM))

	// Encode stego PCM back to MP3 format while preserving original headers
	stegoAudio, err := h.audioDecoder.EncodePCMToMP3(stegoPCM, audioMetadata, audioData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.StegoResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to encode stego audio: %v", err),
		})
		return
	}

	baseFilename := strings.TrimSuffix(audioHeader.Filename, filepath.Ext(audioHeader.Filename))
	outputFilename := fmt.Sprintf("%s_stego.mp3", baseFilename)

	// Set headers for file download
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", outputFilename))
	c.Header("Content-Type", "audio/mpeg")
	c.Header("Content-Length", fmt.Sprintf("%d", len(stegoAudio)))

	// Include PSNR value in custom header for quality assessment
	c.Header("X-Stego-PSNR", fmt.Sprintf("%.2f", psnr))
	c.Header("X-Stego-Message", "Secret message successfully embedded in MP3 file")
	c.Header("X-Stego-Metadata-Preserved", "true")

	c.Data(http.StatusOK, "audio/mpeg", stegoAudio)
}

func (h *StegoHandler) ExtractMessage(c *gin.Context) {
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil { // 32MB limit
		c.JSON(http.StatusBadRequest, models.ExtractResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to parse form: %v", err),
		})
		return
	}

	key := c.PostForm("key")
	useEncryption := c.PostForm("use_encryption") == "true"
	useRandomStart := c.PostForm("use_random_start") == "true"
	lsbBitsStr := c.PostForm("lsb_bits")

	if key == "" {
		c.JSON(http.StatusBadRequest, models.ExtractResponse{
			Success: false,
			Message: "Key is required",
		})
		return
	}

	if err := crypto.ValidateKey(key); err != nil {
		c.JSON(http.StatusBadRequest, models.ExtractResponse{
			Success: false,
			Message: fmt.Sprintf("Invalid key: %v", err),
		})
		return
	}

	lsbBits, err := strconv.Atoi(lsbBitsStr)
	if err != nil || lsbBits < 1 || lsbBits > 4 {
		c.JSON(http.StatusBadRequest, models.ExtractResponse{
			Success: false,
			Message: "LSB bits must be between 1 and 4",
		})
		return
	}

	stegoFile, stegoHeader, err := c.Request.FormFile("stego_file")
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ExtractResponse{
			Success: false,
			Message: "Stego audio file is required",
		})
		return
	}
	defer stegoFile.Close()

	if !isValidMP3File(stegoHeader.Filename) {
		c.JSON(http.StatusBadRequest, models.ExtractResponse{
			Success: false,
			Message: "Invalid audio file format. Only MP3 and WAV files are supported for extraction",
		})
		return
	}

	stegoAudio, err := io.ReadAll(stegoFile)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ExtractResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to read stego audio file: %v", err),
		})
		return
	}

	stegoPCM, _, err := h.audioDecoder.DecodeMP3(stegoAudio)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ExtractResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to decode MP3 stego file: %v", err),
		})
		return
	}

	config := &models.StegoConfig{
		Key:            key,
		UseEncryption:  useEncryption,
		UseRandomStart: useRandomStart,
		LSBBits:        lsbBits,
	}

	// Extraction
	lsbStego := stego.NewLSBSteganography(config)
	secretData, secretFilename, err := lsbStego.Extract(stegoPCM)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ExtractResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to extract secret data: %v", err),
		})
		return
	}

	// Set headers for file download
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", secretFilename))
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Length", fmt.Sprintf("%d", len(secretData)))

	c.Data(http.StatusOK, "application/octet-stream", secretData)
}

func isValidMP3File(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".mp3"
}

func bytesToFloat64(data []byte) []float64 {
	if len(data)%2 != 0 {
		// Handle odd length by ignoring the last byte
		data = data[:len(data)-1]
	}

	samples := make([]float64, len(data)/2)
	for i := range samples {
		// Read little-endian 16-bit sample
		low := int16(data[i*2])
		high := int16(data[i*2+1])
		sample := low | (high << 8)

		// Convert to float64 normalized to [-1.0, 1.0]
		samples[i] = float64(sample) / 32768.0
	}
	return samples
}
