package mp3parser

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// read syncsafe int for ID3v2 size
func syncSafeToInt(b []byte) int {
	return int(b[0]&0x7F)<<21 |
		int(b[1]&0x7F)<<14 |
		int(b[2]&0x7F)<<7 |
		int(b[3]&0x7F)
}

func ReadID3v2(r io.Reader) (*ID3v2Header, []byte, error) {
	buf := make([]byte, 10)
	_, err := io.ReadFull(r, buf)
	if err != nil {
		return nil, nil, err
	}
	if string(buf[:3]) != "ID3" {
		// no ID3v2, seek back
		if seeker, ok := r.(io.Seeker); ok {
			seeker.Seek(-10, io.SeekCurrent)
		}
		return nil, nil, nil
	}
	h := &ID3v2Header{
		Version: [2]byte{buf[3], buf[4]},
		Flags:   buf[5],
		Size:    syncSafeToInt(buf[6:10]),
	}

	// Read the ID3v2 data
	id3Data := make([]byte, h.Size)
	_, err = io.ReadFull(r, id3Data)
	if err != nil {
		return nil, nil, err
	}

	return h, id3Data, nil
}

func ReadFrameHeader(r io.Reader) (*MP3FrameHeader, []byte, []byte, error) {
	headerBytes := make([]byte, 4)
	_, err := io.ReadFull(r, headerBytes)
	if err != nil {
		return nil, nil, nil, err
	}
	header := binary.BigEndian.Uint32(headerBytes)

	// check sync
	if (header & 0xFFE00000) != 0xFFE00000 {
		return nil, nil, nil, fmt.Errorf("invalid sync word: 0x%08X", header)
	}

	versionID := int((header >> 19) & 0x3)
	layer := int((header >> 17) & 0x3)
	prot := ((header >> 16) & 0x1) == 0
	bitrateIdx := int((header >> 12) & 0xF)
	sampleRateIdx := int((header >> 10) & 0x3)
	padding := ((header >> 9) & 0x1) == 1
	channelMode := int((header >> 6) & 0x3)

	// lookup tables (MPEG1 Layer III only for now)
	bitrateTable := [16]int{
		0, 32, 40, 48, 56, 64, 80, 96,
		112, 128, 160, 192, 224, 256, 320, 0,
	}
	sampleRateTable := [4]int{44100, 48000, 32000, 0}

	bitrate := bitrateTable[bitrateIdx] * 1000
	sampleRate := sampleRateTable[sampleRateIdx]

	if bitrate == 0 || sampleRate == 0 {
		return nil, nil, nil, fmt.Errorf("unsupported bitrate or samplerate")
	}

	frameLen := (144*bitrate)/sampleRate + btoi(padding)

	h := &MP3FrameHeader{
		VersionID:     versionID,
		Layer:         layer,
		ProtectionBit: prot,
		Bitrate:       bitrate,
		SampleRate:    sampleRate,
		Padding:       padding,
		ChannelMode:   channelMode,
		FrameLength:   frameLen,
	}

	data := make([]byte, frameLen-4)
	_, err = io.ReadFull(r, data)
	if err != nil {
		return nil, nil, nil, err
	}

	return h, headerBytes, data, nil
}

func ReadID3v1(f *os.File) (*ID3v1Tag, error) {
	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if stat.Size() < 128 {
		return nil, nil
	}
	currentPos, _ := f.Seek(0, io.SeekCurrent)
	f.Seek(-128, io.SeekEnd)
	buf := make([]byte, 128)
	_, err = io.ReadFull(f, buf)
	if err != nil {
		return nil, err
	}
	f.Seek(currentPos, io.SeekStart) // restore position

	if string(buf[:3]) != "TAG" {
		return nil, nil
	}
	return &ID3v1Tag{
		Title:   string(buf[3:33]),
		Artist:  string(buf[33:63]),
		Album:   string(buf[63:93]),
		Year:    string(buf[93:97]),
		Comment: string(buf[97:127]),
		Genre:   buf[127],
	}, nil
}

// ParseMP3File parses an entire MP3 file
func ParseMP3File(data []byte) (*MP3File, error) {
	reader := bytes.NewReader(data)

	mp3File := &MP3File{}

	// Read ID3v2 if present
	id3v2, id3v2Data, err := ReadID3v2(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read ID3v2: %v", err)
	}
	mp3File.ID3v2 = id3v2
	mp3File.ID3v2Data = id3v2Data

	// Read MP3 frames
	for {
		frameHeader, headerBytes, frameData, err := ReadFrameHeader(reader)
		if err != nil {
			if err == io.EOF {
				break
			}
			// Try to skip invalid data and find next frame
			continue
		}

		frame := &MP3Frame{
			Header:      frameHeader,
			HeaderBytes: headerBytes,
			Data:        frameData,
		}
		mp3File.Frames = append(mp3File.Frames, frame)
	}

	return mp3File, nil
}

func WriteMP3File(mp3File *MP3File) ([]byte, error) {
	var buf bytes.Buffer

	// Write ID3v2 if present
	if mp3File.ID3v2 != nil {
		// Write ID3v2 header
		buf.WriteString("ID3")
		buf.WriteByte(mp3File.ID3v2.Version[0])
		buf.WriteByte(mp3File.ID3v2.Version[1])
		buf.WriteByte(mp3File.ID3v2.Flags)

		// Write syncsafe size
		size := mp3File.ID3v2.Size
		sizeBuf := make([]byte, 4)
		sizeBuf[0] = byte((size >> 21) & 0x7F)
		sizeBuf[1] = byte((size >> 14) & 0x7F)
		sizeBuf[2] = byte((size >> 7) & 0x7F)
		sizeBuf[3] = byte(size & 0x7F)
		buf.Write(sizeBuf)

		// Write ID3v2 data
		buf.Write(mp3File.ID3v2Data)
	}

	for _, frame := range mp3File.Frames {
		buf.Write(frame.HeaderBytes)
		buf.Write(frame.Data)
	}

	return buf.Bytes(), nil
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}
