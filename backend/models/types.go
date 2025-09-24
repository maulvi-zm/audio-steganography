// Package models contain needed models
package models

// StegoRequest represents the request for inserting a secret message
type StegoRequest struct {
	Key            string `json:"key" binding:"required"`
	UseEncryption  bool   `json:"use_encryption"`
	UseRandomStart bool   `json:"use_random_start"`
	LSBBits        int    `json:"lsb_bits" binding:"required,min=1,max=4"`
	SecretFilename string `json:"secret_filename"`
}

// StegoResponse represents the response after insertion
type StegoResponse struct {
	Success      bool    `json:"success"`
	Message      string  `json:"message"`
	PSNR         float64 `json:"psnr,omitempty"`
	StegoFileURL string  `json:"stego_file_url,omitempty"`
}

// ExtractRequest represents the request for extracting a secret message
type ExtractRequest struct {
	Key            string `json:"key" binding:"required"`
	UseEncryption  bool   `json:"use_encryption"`
	UseRandomStart bool   `json:"use_random_start"`
	LSBBits        int    `json:"lsb_bits" binding:"required,min=1,max=4"`
}

// ExtractResponse represents the response after extraction
type ExtractResponse struct {
	Success        bool   `json:"success"`
	Message        string `json:"message"`
	SecretFileURL  string `json:"secret_file_url,omitempty"`
	SecretFilename string `json:"secret_filename,omitempty"`
}

// AudioMetadata represents metadata about an audio file
type AudioMetadata struct {
	SampleRate int
	Channels   int
	BitDepth   int
	Duration   float64
	TotalBytes int
}

// StegoConfig represents configuration for steganography operations
type StegoConfig struct {
	Key            string
	UseEncryption  bool
	UseRandomStart bool
	LSBBits        int
	SecretFilename string
}
