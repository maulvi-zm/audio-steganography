// Package mp3parser to parse MP3
package mp3parser

import (
	"fmt"
	"io"
	"math"
)

type MP3FrameRegions struct {
	SideInfo      []byte
	MainData      []byte
	AncillaryData []byte
	Padding       []byte
}

type BitReader struct {
	data []byte
	pos  int // bit position
}

func NewBitReader(data []byte) *BitReader {
	return &BitReader{data: data}
}

func (br *BitReader) ReadBits(n int) (uint32, error) {
	if n <= 0 || n > 32 {
		return 0, fmt.Errorf("invalid bit count")
	}
	var val uint32
	for range n {
		bytePos := br.pos / 8
		if bytePos >= len(br.data) {
			return 0, io.EOF
		}
		bitPos := 7 - (br.pos % 8)
		bit := (br.data[bytePos] >> bitPos) & 1
		val = (val << 1) | uint32(bit)
		br.pos++
	}
	return val, nil
}

type GranuleChannelInfo struct {
	Part23Length uint32
	BigValues    uint32
	GlobalGain   uint32
}

func ParseSideInfo(frameHeader *MP3FrameHeader, sideInfo []byte) ([][]GranuleChannelInfo, error) {
	br := NewBitReader(sideInfo)

	if frameHeader.VersionID == 3 { // MPEG-1
		_, _ = br.ReadBits(9)
	} else {
		_, _ = br.ReadBits(8)
	}

	// Skip private bits
	if frameHeader.VersionID == 3 { // MPEG-1
		if frameHeader.ChannelMode == 3 { // Mono
			_, _ = br.ReadBits(5)
		} else {
			_, _ = br.ReadBits(3)
		}
	} else {
		if frameHeader.ChannelMode == 3 { // Mono
			_, _ = br.ReadBits(1)
		} else {
			_, _ = br.ReadBits(2)
		}
	}

	// Granule count: MPEG-1 = 2, MPEG-2/2.5 = 1
	granules := 1
	if frameHeader.VersionID == 3 {
		granules = 2
	}
	channels := 2
	if frameHeader.ChannelMode == 3 {
		channels = 1
	}

	result := make([][]GranuleChannelInfo, granules)
	for gr := 0; gr < granules; gr++ {
		result[gr] = make([]GranuleChannelInfo, channels)
		for ch := 0; ch < channels; ch++ {
			p23, _ := br.ReadBits(12)
			bv, _ := br.ReadBits(9)
			gg, _ := br.ReadBits(8)
			// (skip other fields if not needed)
			result[gr][ch] = GranuleChannelInfo{
				Part23Length: p23,
				BigValues:    bv,
				GlobalGain:   gg,
			}
		}
	}
	return result, nil
}

func AnalyzeFrameData(frameHeader *MP3FrameHeader, frameData []byte) (*MP3FrameRegions, error) {
	if len(frameData) < 4 {
		return nil, fmt.Errorf("frame data too short")
	}

	regions := &MP3FrameRegions{}

	// Calculate side info size
	var sideInfoSize int
	if frameHeader.VersionID == 3 { // MPEG-1
		if frameHeader.ChannelMode == 3 {
			sideInfoSize = 17
		} else {
			sideInfoSize = 32
		}
	} else { // MPEG-2/2.5
		if frameHeader.ChannelMode == 3 {
			sideInfoSize = 9
		} else {
			sideInfoSize = 17
		}
	}

	if sideInfoSize >= len(frameData) {
		regions.SideInfo = frameData
		return regions, nil
	}

	// Split side info
	regions.SideInfo = frameData[:sideInfoSize]
	remaining := frameData[sideInfoSize:]

	// Parse side info → get part2_3_length[]
	granules, err := ParseSideInfo(frameHeader, regions.SideInfo)
	if err != nil {
		return nil, err
	}

	// Sum part2_3_length (bits → bytes)
	mainBits := 0
	for _, gr := range granules {
		for _, ch := range gr {
			mainBits += int(ch.Part23Length)
		}
	}
	mainBytes := int(math.Ceil(float64(mainBits) / 8.0))

	// Safety bounds: ensure we don't exceed frame data and leave some space for ancillary
	maxMainBytes := len(remaining) - 20 // Reserve at least 20 bytes for ancillary/padding
	if maxMainBytes < 0 {
		maxMainBytes = len(remaining)
	}
	if mainBytes > maxMainBytes {
		mainBytes = maxMainBytes
	}

	// Assign regions
	regions.MainData = remaining[:mainBytes]
	rest := remaining[mainBytes:]

	// Split ancillary vs padding
	paddingStart := len(rest)
	for i := len(rest) - 1; i >= 0; i-- {
		if rest[i] == 0x00 {
			paddingStart = i
		} else {
			break
		}
	}

	regions.AncillaryData = rest[:paddingStart]
	regions.Padding = rest[paddingStart:]

	return regions, nil
}

func (regions *MP3FrameRegions) GetSafeModificationBytes() []byte {
	safe := make([]byte, 0)
	safe = append(safe, regions.AncillaryData...)
	safe = append(safe, regions.Padding...)
	return safe
}

func (regions *MP3FrameRegions) ReconstructFrameData(modifiedSafe []byte) []byte {
	frame := make([]byte, 0)
	frame = append(frame, regions.SideInfo...)
	frame = append(frame, regions.MainData...)

	ancLen := len(regions.AncillaryData)
	padLen := len(regions.Padding)

	if len(modifiedSafe) <= ancLen+padLen {
		frame = append(frame, modifiedSafe...)
		if len(modifiedSafe) < ancLen+padLen {
			// Fill remaining with zeros (keep frame size stable)
			frame = append(frame, make([]byte, ancLen+padLen-len(modifiedSafe))...)
		}
	} else {
		frame = append(frame, modifiedSafe[:ancLen+padLen]...)
	}
	return frame
}
