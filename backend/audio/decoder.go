package audio

import (
	"fmt"
	"steganography-backend/models"
	"steganography-backend/mp3parser"

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

// AnalyzeMP3 analyzes MP3 file structure and returns basic metadata
func (ad *AudioDecoder) AnalyzeMP3(mp3Data []byte) (*MP3Info, error) {
	mp3File, err := mp3parser.ParseMP3File(mp3Data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse MP3: %v", err)
	}

	if len(mp3File.Frames) == 0 {
		return nil, fmt.Errorf("no MP3 frames found")
	}

	// Use first frame for metadata
	firstFrame := mp3File.Frames[0]

	totalFrames := len(mp3File.Frames)
	totalDataBytes := 0
	for _, frame := range mp3File.Frames {
		totalDataBytes += len(frame.Data)
	}

	return &MP3Info{
		Bitrate:        firstFrame.Header.Bitrate,
		SampleRate:     firstFrame.Header.SampleRate,
		ChannelMode:    firstFrame.Header.ChannelMode,
		TotalFrames:    totalFrames,
		TotalDataBytes: totalDataBytes,
		HasID3v1:       mp3File.ID3v1 != nil,
		HasID3v2:       mp3File.ID3v2 != nil,
	}, nil
}

// DecodeMP3ToPCM decodes MP3 data to PCM for PSNR calculation
func (ad *AudioDecoder) DecodeMP3ToPCM(mp3Data []byte) ([]byte, *models.AudioMetadata, error) {
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

	return data, metadata, nil
}

// MP3Info contains information about an MP3 file
type MP3Info struct {
	Bitrate        int
	SampleRate     int
	ChannelMode    int
	TotalFrames    int
	TotalDataBytes int
	HasID3v1       bool
	HasID3v2       bool
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

func (ad *AudioDecoder) ValidateAudioCapacity(pcmData []byte, secretData []byte, filename string, lsbBits int) error {
	maxCapacity := ad.CalculateMaxSecretLength(pcmData, lsbBits)
	requiredCapacity := len(secretData) + len(filename)

	if requiredCapacity > maxCapacity {
		return fmt.Errorf("insufficient audio capacity: required %d bytes, available %d bytes",
			requiredCapacity, maxCapacity)
	}

	return nil
}
