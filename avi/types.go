package avi

import (
	"io"
	"time"
)

// StreamType represents the type of media stream
type StreamType string

const (
	StreamTypeVideo StreamType = "video"
	StreamTypeAudio StreamType = "audio"
)

// Codec represents codec information
type Codec struct {
	Name    string
	FourCC  [4]byte
	Type    StreamType
	Width   int // for video
	Height  int // for video
	FPS     float64 // for video
	Channels int // for audio
	SampleRate int // for audio
	BitDepth int // for audio
}

// Packet represents a single media packet
type Packet struct {
	StreamIndex int
	Codec       StreamType
	Data        []byte
	PTS         int64     // presentation timestamp
	DTS         int64     // decode timestamp
	Duration    int64
	Size        int
	Position    int64     // position in file
	Flags       string    // keyframe flags
	PTSTime     time.Duration
	DTSTime     time.Duration
	DurationTime time.Duration
}

// Stream represents a media stream
type Stream struct {
	Index     int
	Type      StreamType
	Codec     Codec
	Duration  time.Duration
	PacketCount int
}

// FileInfo contains metadata about the AVI file
type FileInfo struct {
	Filename    string
	Duration    time.Duration
	FileSize    int64
	Streams     []Stream
	VideoStreams int
	AudioStreams int
}

// Demuxer interface for reading AVI files
type Demuxer interface {
	// Open opens an AVI reader
	Open(r io.ReadSeeker, size int64) error
	
	// OpenFile opens an AVI file for reading (convenience method)
	OpenFile(filename string) error
	
	// GetFileInfo returns metadata about the file
	GetFileInfo() (*FileInfo, error)
	
	// GetStreams returns all streams in the file
	GetStreams() ([]Stream, error)
	
	// ReadPacket reads the next packet from the file
	ReadPacket() (*Packet, error)
	
	// Seek seeks to a specific timestamp
	Seek(timestamp time.Duration) error
	
	// Close closes the reader
	Close() error
}

// Muxer interface for writing AVI files
type Muxer interface {
	// Create creates a new AVI writer
	Create(w io.WriteSeeker) error
	
	// CreateFile creates a new AVI file for writing (convenience method)
	CreateFile(filename string) error
	
	// AddStream adds a new stream to the file
	AddStream(codec Codec) (int, error)
	
	// WritePacket writes a packet to the file
	WritePacket(packet *Packet) error
	
	// Finalize finalizes the file (writes headers, indices)
	Finalize() error
	
	// Close closes the writer
	Close() error
}

// Reader wraps an io.ReadSeeker for AVI reading
type Reader struct {
	r io.ReadSeeker
	filename string
	fileSize int64
	streams []Stream
	fileInfo *FileInfo
	moviOffset int64 // Offset to movi chunk data
	indexEntries []IndexEntry // Index entries for seeking
}

// Writer wraps an io.WriteSeeker for AVI writing  
type Writer struct {
	w io.WriteSeeker
	filename string
	streams []Stream
	packets []Packet
}