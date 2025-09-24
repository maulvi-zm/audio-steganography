package audio

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"steganography-backend/models"

	"github.com/bogem/id3v2"
	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"github.com/tosone/minimp3"
)

const (
	FilenameLengthBytes        = 4
	DataLengthBytes            = 4
	MaximumFilenameBytesLength = 50
	BitsInByte                 = 8
)

type AudioDecoder struct{}

func NewAudioDecoder() *AudioDecoder {
	return &AudioDecoder{}
}

func (ad *AudioDecoder) DecodeMP3(mp3Data []byte) ([]byte, *models.AudioMetadata, error) {
	decoder, data, err := minimp3.DecodeFull(mp3Data)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode MP3: %v", err)
	}
	defer decoder.Close()

	totalBytes := len(data)
	samplesPerChannel := totalBytes / 2 / decoder.Channels // 2 bytes per 16-bit sample
	duration := float64(samplesPerChannel) / float64(decoder.SampleRate)

	metadata := &models.AudioMetadata{
		SampleRate: decoder.SampleRate,
		Channels:   decoder.Channels,
		BitDepth:   16,
		Duration:   duration,
		TotalBytes: totalBytes,
	}

	// The data is already in byte format from minimp3
	return data, metadata, nil
}

func (ad *AudioDecoder) CalculateMaxSecretLength(pcmData []byte, lsbBits int) int {
	bitsPerByte := lsbBits
	totalBits := len(pcmData) * bitsPerByte
	totalBytes := totalBits / BitsInByte

	metadataBytes := FilenameLengthBytes + DataLengthBytes
	filenameBytes := MaximumFilenameBytesLength

	maxSecretBytes := totalBytes - metadataBytes - filenameBytes
	if maxSecretBytes < 0 {
		return 0
	}

	return maxSecretBytes
}

func (ad *AudioDecoder) EncodePCMToWAV(pcmData []byte, metadata *models.AudioMetadata) ([]byte, error) {
	if len(pcmData)%2 != 0 {
		return nil, fmt.Errorf("PCM data length must be even for 16-bit samples")
	}

	sampleCount := len(pcmData) / 2
	samples := make([]int, sampleCount)

	for i := range sampleCount {
		// Little-endian 16-bit sample
		low := int16(pcmData[i*2])
		high := int16(pcmData[i*2+1])
		sample := int(low | (high << 8))

		if sample > 32767 {
			sample = sample - 65536
		}
		samples[i] = sample
	}

	// Create audio buffer
	format := &audio.Format{
		NumChannels: metadata.Channels,
		SampleRate:  metadata.SampleRate,
	}

	buf := &audio.IntBuffer{
		Format: format,
		Data:   samples,
	}

	// Create a temporary file for WAV encoding since wav.NewEncoder needs WriteSeeker
	tempFile, err := os.CreateTemp("", "temp_*.wav")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %v", err)
	}
	defer tempFile.Close()
	defer os.Remove(tempFile.Name())

	encoder := wav.NewEncoder(tempFile, metadata.SampleRate, metadata.BitDepth, metadata.Channels, 1)

	if err := encoder.Write(buf); err != nil {
		return nil, fmt.Errorf("failed to encode WAV: %v", err)
	}

	if err := encoder.Close(); err != nil {
		return nil, fmt.Errorf("failed to close WAV encoder: %v", err)
	}

	// Read the file content back
	tempFile.Seek(0, 0)
	wavData, err := io.ReadAll(tempFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read WAV data: %v", err)
	}

	return wavData, nil
}

func (ad *AudioDecoder) ValidateAudioCapacity(pcmData []byte, secretData []byte, filename string, lsbBits int) error {
	maxCapacity := ad.CalculateMaxSecretLength(pcmData, lsbBits)
	requiredCapacity := len(secretData) + len(filename)

	if requiredCapacity > maxCapacity {
		return fmt.Errorf("insufficient audio capacity: required %d bytes, available %d bytes",
			requiredCapacity, maxCapacity)
	}

	return nil
}

func (ad *AudioDecoder) EncodePCMToMP3(pcmData []byte, metadata *models.AudioMetadata, originalMP3Data []byte) ([]byte, error) {
	// Create temporary WAV file for LAME encoding
	tempWAV, err := os.CreateTemp("", "temp_*.wav")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary WAV file: %v", err)
	}
	defer os.Remove(tempWAV.Name())
	defer tempWAV.Close()

	tempMP3, err := os.CreateTemp("", "temp_*.mp3")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary MP3 file: %v", err)
	}
	defer os.Remove(tempMP3.Name())
	defer tempMP3.Close()

	// Convert PCM data to WAV format first
	wavData, err := ad.EncodePCMToWAV(pcmData, metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to encode PCM to WAV: %v", err)
	}

	// Write WAV data to temporary file
	if _, err := tempWAV.Write(wavData); err != nil {
		return nil, fmt.Errorf("failed to write WAV data: %v", err)
	}
	tempWAV.Close()

	cmd := exec.Command("lame", "--preset", "standard", "-h", "-q", "0", "--add-id3v2", "--pad-id3v2", "--nohist", tempWAV.Name(), tempMP3.Name())
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to encode MP3 (lame not installed?): %v", err)
	}

	mp3Data, err := os.ReadFile(tempMP3.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to read MP3 file: %v", err)
	}

	// Preserve metadata from original file
	mp3DataWithMeta, err := ad.preserveMP3Metadata(originalMP3Data, mp3Data)
	if err != nil {
		fmt.Printf("Warning: Could not preserve metadata: %v\n", err)
		fmt.Printf("Returning steganographed MP3 without original metadata\n")
		return mp3Data, nil
	}

	return mp3DataWithMeta, nil
}

func (ad *AudioDecoder) preserveMP3Metadata(originalMP3Data, newMP3Data []byte) ([]byte, error) {
	tempOriginal, err := os.CreateTemp("", "original_*.mp3")
	if err != nil {
		return newMP3Data, fmt.Errorf("failed to create temp original file: %v", err)
	}
	defer func() {
		tempOriginal.Close()
		os.Remove(tempOriginal.Name())
	}()

	tempNew, err := os.CreateTemp("", "new_*.mp3")
	if err != nil {
		return newMP3Data, fmt.Errorf("failed to create temp new file: %v", err)
	}
	defer func() {
		tempNew.Close()
		os.Remove(tempNew.Name())
	}()

	if _, err := tempOriginal.Write(originalMP3Data); err != nil {
		return newMP3Data, fmt.Errorf("failed to write original data: %v", err)
	}
	if _, err := tempNew.Write(newMP3Data); err != nil {
		return newMP3Data, fmt.Errorf("failed to write new data: %v", err)
	}

	// Close files to ensure data is written
	tempOriginal.Close()
	tempNew.Close()

	originalTag, err := id3v2.Open(tempOriginal.Name(), id3v2.Options{Parse: true})
	if err != nil {
		fmt.Printf("Warning: Could not parse original metadata: %v\n", err)
		return newMP3Data, nil
	}

	newTag, err := id3v2.Open(tempNew.Name(), id3v2.Options{Parse: true})
	if err != nil {
		fmt.Printf("Warning: Could not parse original metadata: %v\n", err)
		return newMP3Data, nil
	}

	// Set new values for common tags
	newTag.SetTitle(originalTag.Title())
	newTag.SetArtist(originalTag.Artist())
	newTag.SetAlbum(originalTag.Album())
	newTag.SetGenre(originalTag.Genre())
	newTag.SetYear(originalTag.Year())

	if err := newTag.Save(); err != nil {
		fmt.Printf("Warning: Could not save metadata to stego file: %v\n", err)
		return newMP3Data, nil
	}

	updatedData, err := os.ReadFile(tempNew.Name())
	if err != nil {
		return newMP3Data, fmt.Errorf("failed to read updated MP3 data: %v", err)
	}

	fmt.Printf("Successfully preserved MP3 metadata in steganographed file\n")
	return updatedData, nil
}
