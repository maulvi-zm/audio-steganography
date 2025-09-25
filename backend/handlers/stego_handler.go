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

	// Analyze MP3 structure
	mp3Info, err := h.audioDecoder.AnalyzeMP3(audioData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.StegoResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to analyze MP3 file: %v", err),
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

	mp3Stego := stego.NewMP3LSBSteganography(config)
	capacity, err := mp3Stego.CalculateCapacity(audioData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.StegoResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to calculate capacity: %v", err),
		})
		return
	}

	if len(secretData)+len(secretHeader.Filename)+8 > capacity {
		c.JSON(http.StatusBadRequest, models.StegoResponse{
			Success: false,
			Message: fmt.Sprintf("Secret data too large. Maximum capacity: %d bytes, required: %d bytes",
				capacity, len(secretData)+len(secretHeader.Filename)+8),
		})
		return
	}

	// Embed secret data directly into MP3 bitstream
	stegoAudio, err := mp3Stego.EmbedInMP3(audioData, secretData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.StegoResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to embed secret data: %v", err),
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

	// Include metadata about the steganography operation
	c.Header("X-Stego-Method", "MP3 Bitstream LSB")
	c.Header("X-Stego-Message", "Secret message successfully embedded directly in MP3 bitstream")
	c.Header("X-Stego-Capacity", fmt.Sprintf("%d", capacity))
	c.Header("X-Stego-Frames", fmt.Sprintf("%d", mp3Info.TotalFrames))

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

	config := &models.StegoConfig{
		Key:            key,
		UseEncryption:  useEncryption,
		UseRandomStart: useRandomStart,
		LSBBits:        lsbBits,
	}

	// Extract directly from MP3 bitstream
	mp3Stego := stego.NewMP3LSBSteganography(config)
	secretData, secretFilename, err := mp3Stego.ExtractFromMP3(stegoAudio)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ExtractResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to extract secret data: %v", err),
		})
		return
	}

	if len(secretData) == 0 {
		c.JSON(http.StatusInternalServerError, models.ExtractResponse{
			Success: false,
			Message: "No secret data extracted. Possible causes: (1) File contains no embedded data, (2) Wrong extraction parameters (key, LSB bits, encryption, random start), (3) MP3 file structure was modified after embedding.",
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
