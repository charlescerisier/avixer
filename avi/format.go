package avi

import (
	"encoding/binary"
	"fmt"
)

// AVI Format Constants
const (
	// RIFF chunk identifiers
	RIFFSignature = "RIFF"
	AVISignature  = "AVI "
	LISTSignature = "LIST"
	
	// AVI List types
	HDRLList = "hdrl"
	STRLList = "strl" 
	MOVIList = "movi"
	
	// Chunk types
	AVIHChunk = "avih"
	STRHChunk = "strh"
	STRFChunk = "strf"
	STRDChunk = "strd"
	STRNChunk = "strn"
	INDXChunk = "indx"
	IDX1Chunk = "idx1"
	
	// Stream types
	STREAMTypeVideo = "vids"
	STREAMTypeAudio = "auds"
	STREAMTypeText  = "txts"
	
	// Video codecs (common ones)
	CODECMjpeg = "MJPG"
	CODECMP4V  = "MP4V"
	CODECH264  = "H264"
	CODECXVID  = "XVID"
	CODECDIVX  = "DIVX"
)

// RIFFHeader represents the main RIFF header
type RIFFHeader struct {
	Signature [4]byte // "RIFF"
	FileSize  uint32  // File size minus 8 bytes
	Type      [4]byte // "AVI "
}

// ChunkHeader represents a generic chunk header
type ChunkHeader struct {
	ID   [4]byte // Chunk identifier
	Size uint32  // Chunk data size
}

// LISTHeader represents a LIST chunk header
type LISTHeader struct {
	ChunkHeader
	Type [4]byte // List type
}

// AVIMainHeader represents the main AVI header (avih chunk)
type AVIMainHeader struct {
	MicroSecPerFrame    uint32 // Frame display rate
	MaxBytesPerSec      uint32 // Maximum data rate
	PaddingGranularity  uint32 // Data alignment
	Flags               uint32 // File flags
	TotalFrames         uint32 // Total number of frames
	InitialFrames       uint32 // Initial frames for interleaved files
	Streams             uint32 // Number of streams
	SuggestedBufferSize uint32 // Suggested buffer size
	Width               uint32 // Video width
	Height              uint32 // Video height
	Reserved            [4]uint32 // Reserved fields
}

// AVIStreamHeader represents a stream header (strh chunk)
type AVIStreamHeader struct {
	Type                [4]byte // Stream type (vids, auds, etc.)
	Handler             [4]byte // Codec handler
	Flags               uint32  // Stream flags
	Priority            uint16  // Stream priority
	Language            uint16  // Language
	InitialFrames       uint32  // Initial frames
	Scale               uint32  // Time scale
	Rate                uint32  // Sample rate
	Start               uint32  // Start time
	Length              uint32  // Stream length
	SuggestedBufferSize uint32  // Suggested buffer size
	Quality             uint32  // Quality
	SampleSize          uint32  // Sample size
	Frame               struct{ // Frame rectangle
		Left   uint16
		Top    uint16
		Right  uint16
		Bottom uint16
	}
}

// BitmapInfoHeader represents video format info
type BitmapInfoHeader struct {
	Size          uint32  // Structure size
	Width         int32   // Image width
	Height        int32   // Image height  
	Planes        uint16  // Number of planes
	BitCount      uint16  // Bits per pixel
	Compression   [4]byte // Compression type
	SizeImage     uint32  // Image size
	XPelsPerMeter int32   // Horizontal resolution
	YPelsPerMeter int32   // Vertical resolution
	ClrUsed       uint32  // Colors used
	ClrImportant  uint32  // Important colors
}

// WaveFormatEx represents audio format info
type WaveFormatEx struct {
	FormatTag      uint16 // Audio format
	Channels       uint16 // Number of channels
	SamplesPerSec  uint32 // Sample rate
	AvgBytesPerSec uint32 // Average bytes per second
	BlockAlign     uint16 // Block alignment
	BitsPerSample  uint16 // Bits per sample
	Size           uint16 // Extra format bytes
}

// IndexEntry represents an index entry (idx1)
type IndexEntry struct {
	ChunkID [4]byte // Chunk identifier
	Flags   uint32  // Flags
	Offset  uint32  // Offset in file
	Size    uint32  // Chunk size
}

// Helper functions for chunk operations
func MakeChunkID(streamIndex int, twoCC string) [4]byte {
	var id [4]byte
	id[0] = byte('0' + (streamIndex / 10))
	id[1] = byte('0' + (streamIndex % 10))
	id[2] = twoCC[0]
	id[3] = twoCC[1]
	return id
}

func ChunkIDToString(id [4]byte) string {
	return string(id[:])
}

func StringToChunkID(s string) [4]byte {
	var id [4]byte
	copy(id[:], s)
	return id
}

// Read/Write helpers for binary data
func ReadChunkHeader(data []byte) ChunkHeader {
	var header ChunkHeader
	copy(header.ID[:], data[0:4])
	header.Size = binary.LittleEndian.Uint32(data[4:8])
	return header
}

func WriteChunkHeader(header ChunkHeader) []byte {
	data := make([]byte, 8)
	copy(data[0:4], header.ID[:])
	binary.LittleEndian.PutUint32(data[4:8], header.Size)
	return data
}

func AlignSize(size uint32) uint32 {
	return (size + 1) &^ 1 // Align to even boundary
}

// Validation functions
func IsValidRIFFSignature(sig [4]byte) bool {
	return string(sig[:]) == RIFFSignature
}

func IsValidAVISignature(sig [4]byte) bool {
	return string(sig[:]) == AVISignature
}

func IsVideoStream(streamType [4]byte) bool {
	return string(streamType[:]) == STREAMTypeVideo
}

func IsAudioStream(streamType [4]byte) bool {
	return string(streamType[:]) == STREAMTypeAudio
}

// Error types
type AVIError struct {
	Op  string
	Err error
}

func (e *AVIError) Error() string {
	return fmt.Sprintf("avi: %s: %v", e.Op, e.Err)
}

func (e *AVIError) Unwrap() error {
	return e.Err
}