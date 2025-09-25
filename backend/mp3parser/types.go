package mp3parser

// ID3v2Header represents ID3v2 tag header
type ID3v2Header struct {
	Version [2]byte
	Flags   byte
	Size    int
}

// MP3FrameHeader represents an MP3 frame header
type MP3FrameHeader struct {
	VersionID     int
	Layer         int
	ProtectionBit bool
	Bitrate       int
	SampleRate    int
	Padding       bool
	ChannelMode   int
	FrameLength   int
}

// ID3v1Tag represents ID3v1 tag (128 bytes at end of file)
type ID3v1Tag struct {
	Title   string
	Artist  string
	Album   string
	Year    string
	Comment string
	Genre   byte
}

// MP3Frame represents a complete MP3 frame
type MP3Frame struct {
	Header      *MP3FrameHeader
	HeaderBytes []byte // Original 4-byte header - NEVER MODIFY
	Data        []byte // Frame payload data - steganography goes here
}

// MP3File represents the structure of an MP3 file
type MP3File struct {
	ID3v2     *ID3v2Header
	ID3v2Data []byte
	Frames    []*MP3Frame
	ID3v1     *ID3v1Tag
}
