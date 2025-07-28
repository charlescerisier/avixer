package avi

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// NewMuxer creates a new AVI muxer
func NewMuxer() Muxer {
	return &Writer{}
}

// Create creates a new AVI writer
func (w *Writer) Create(writer io.WriteSeeker) error {
	w.w = writer
	w.filename = "" // No filename when using writer directly
	w.streams = nil
	w.packets = nil

	return nil
}

// CreateFile creates a new AVI file for writing (convenience method)
func (w *Writer) CreateFile(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return &AVIError{Op: "create", Err: err}
	}

	w.filename = filename
	
	// Use the generic Create method
	return w.Create(file)
}

// AddStream adds a new stream to the file
func (w *Writer) AddStream(codec Codec) (int, error) {
	if w.w == nil {
		return -1, &AVIError{Op: "add stream", Err: fmt.Errorf("file not created")}
	}

	stream := Stream{
		Index: len(w.streams),
		Type:  codec.Type,
		Codec: codec,
	}

	w.streams = append(w.streams, stream)
	return stream.Index, nil
}

// WritePacket writes a packet to the file
func (w *Writer) WritePacket(packet *Packet) error {
	if w.w == nil {
		return &AVIError{Op: "write packet", Err: fmt.Errorf("file not created")}
	}

	if packet.StreamIndex >= len(w.streams) {
		return &AVIError{Op: "write packet", Err: fmt.Errorf("invalid stream index")}
	}

	// Store packet for later writing
	w.packets = append(w.packets, *packet)
	return nil
}

// Finalize finalizes the file (writes headers, indices)
func (w *Writer) Finalize() error {
	if w.w == nil {
		return &AVIError{Op: "finalize", Err: fmt.Errorf("file not created")}
	}

	// Write the complete AVI structure
	if err := w.writeAVIFile(); err != nil {
		return err
	}

	return nil
}

// writeAVIFile writes the complete AVI file structure
func (w *Writer) writeAVIFile() error {
	// Calculate file size (we'll update this later)
	moviSize := w.calculateMOVISize()
	hdrlSize := w.calculateHDRLSize()
	idx1Size := w.calculateIDX1Size()
	
	totalSize := 4 + hdrlSize + 8 + moviSize + 8 + idx1Size // AVI signature + hdrl + movi header + movi data + idx1 header + idx1 data

	// Write RIFF header
	riffHeader := RIFFHeader{
		Signature: StringToChunkID(RIFFSignature),
		FileSize:  uint32(totalSize),
		Type:      StringToChunkID(AVISignature),
	}

	if err := binary.Write(w.w, binary.LittleEndian, &riffHeader); err != nil {
		return &AVIError{Op: "write riff header", Err: err}
	}

	// Write hdrl LIST
	if err := w.writeHDRLList(); err != nil {
		return err
	}

	// Write movi LIST
	if err := w.writeMOVIList(); err != nil {
		return err
	}

	// Write idx1 chunk
	if err := w.writeIDX1Chunk(); err != nil {
		return err
	}

	return nil
}

// writeHDRLList writes the header list
func (w *Writer) writeHDRLList() error {
	hdrlSize := w.calculateHDRLSize()

	// Write LIST header
	listHeader := LISTHeader{
		ChunkHeader: ChunkHeader{
			ID:   StringToChunkID(LISTSignature),
			Size: hdrlSize,
		},
		Type: StringToChunkID(HDRLList),
	}

	if err := binary.Write(w.w, binary.LittleEndian, &listHeader); err != nil {
		return &AVIError{Op: "write hdrl list", Err: err}
	}

	// Write avih chunk
	if err := w.writeAVIHChunk(); err != nil {
		return err
	}

	// Write strl for each stream
	for i := range w.streams {
		if err := w.writeSTRLList(i); err != nil {
			return err
		}
	}

	return nil
}

// writeAVIHChunk writes the main AVI header
func (w *Writer) writeAVIHChunk() error {
	// Calculate values
	var totalFrames uint32
	var maxBytesPerSec uint32
	var microSecPerFrame uint32
	var width, height uint32

	// Find video stream for dimensions and frame rate
	for _, stream := range w.streams {
		if stream.Type == StreamTypeVideo {
			width = uint32(stream.Codec.Width)
			height = uint32(stream.Codec.Height)
			if stream.Codec.FPS > 0 {
				microSecPerFrame = uint32(1000000.0 / stream.Codec.FPS)
			}
			break
		}
	}

	// Count frames
	for _, packet := range w.packets {
		if w.streams[packet.StreamIndex].Type == StreamTypeVideo {
			totalFrames++
		}
	}

	header := AVIMainHeader{
		MicroSecPerFrame:    microSecPerFrame,
		MaxBytesPerSec:      maxBytesPerSec,
		PaddingGranularity:  0,
		Flags:               0x810, // AVIF_HASINDEX | AVIF_ISINTERLEAVED
		TotalFrames:         totalFrames,
		InitialFrames:       0,
		Streams:             uint32(len(w.streams)),
		SuggestedBufferSize: 0,
		Width:               width,
		Height:              height,
		Reserved:            [4]uint32{0, 0, 0, 0},
	}

	// Write chunk header
	chunkHeader := ChunkHeader{
		ID:   StringToChunkID(AVIHChunk),
		Size: 56, // sizeof(AVIMainHeader)
	}

	if err := binary.Write(w.w, binary.LittleEndian, &chunkHeader); err != nil {
		return &AVIError{Op: "write avih header", Err: err}
	}

	if err := binary.Write(w.w, binary.LittleEndian, &header); err != nil {
		return &AVIError{Op: "write avih", Err: err}
	}

	return nil
}

// writeSTRLList writes a stream list
func (w *Writer) writeSTRLList(streamIndex int) error {
	strlSize := w.calculateSTRLSize(streamIndex)

	// Write LIST header
	listHeader := LISTHeader{
		ChunkHeader: ChunkHeader{
			ID:   StringToChunkID(LISTSignature),
			Size: strlSize,
		},
		Type: StringToChunkID(STRLList),
	}

	if err := binary.Write(w.w, binary.LittleEndian, &listHeader); err != nil {
		return &AVIError{Op: "write strl list", Err: err}
	}

	// Write strh chunk
	if err := w.writeSTRHChunk(streamIndex); err != nil {
		return err
	}

	// Write strf chunk
	if err := w.writeSTRFChunk(streamIndex); err != nil {
		return err
	}

	return nil
}

// writeSTRHChunk writes a stream header
func (w *Writer) writeSTRHChunk(streamIndex int) error {
	stream := w.streams[streamIndex]

	var streamType [4]byte
	if stream.Type == StreamTypeVideo {
		streamType = StringToChunkID(STREAMTypeVideo)
	} else if stream.Type == StreamTypeAudio {
		streamType = StringToChunkID(STREAMTypeAudio)
	}

	// Calculate scale and rate
	var scale, rate uint32 = 1, 1
	if stream.Type == StreamTypeVideo && stream.Codec.FPS > 0 {
		scale = 1000
		rate = uint32(stream.Codec.FPS * 1000)
	} else if stream.Type == StreamTypeAudio && stream.Codec.SampleRate > 0 {
		scale = 1
		rate = uint32(stream.Codec.SampleRate)
	}

	// Count packets for this stream
	var length uint32
	for _, packet := range w.packets {
		if packet.StreamIndex == streamIndex {
			length++
		}
	}

	header := AVIStreamHeader{
		Type:                streamType,
		Handler:             stream.Codec.FourCC,
		Flags:               0,
		Priority:            0,
		Language:            0,
		InitialFrames:       0,
		Scale:               scale,
		Rate:                rate,
		Start:               0,
		Length:              length,
		SuggestedBufferSize: 0,
		Quality:             0xFFFFFFFF,
		SampleSize:          0,
	}

	// Set frame rectangle for video
	if stream.Type == StreamTypeVideo {
		header.Frame.Left = 0
		header.Frame.Top = 0
		header.Frame.Right = uint16(stream.Codec.Width)
		header.Frame.Bottom = uint16(stream.Codec.Height)
	}

	// Write chunk header
	chunkHeader := ChunkHeader{
		ID:   StringToChunkID(STRHChunk),
		Size: 56, // sizeof(AVIStreamHeader)
	}

	if err := binary.Write(w.w, binary.LittleEndian, &chunkHeader); err != nil {
		return &AVIError{Op: "write strh header", Err: err}
	}

	if err := binary.Write(w.w, binary.LittleEndian, &header); err != nil {
		return &AVIError{Op: "write strh", Err: err}
	}

	return nil
}

// writeSTRFChunk writes stream format chunk
func (w *Writer) writeSTRFChunk(streamIndex int) error {
	stream := w.streams[streamIndex]

	if stream.Type == StreamTypeVideo {
		return w.writeVideoFormat(streamIndex)
	} else if stream.Type == StreamTypeAudio {
		return w.writeAudioFormat(streamIndex)
	}

	return nil
}

// writeVideoFormat writes video format info
func (w *Writer) writeVideoFormat(streamIndex int) error {
	stream := w.streams[streamIndex]

	bih := BitmapInfoHeader{
		Size:          40, // sizeof(BitmapInfoHeader)
		Width:         int32(stream.Codec.Width),
		Height:        int32(stream.Codec.Height),
		Planes:        1,
		BitCount:      24, // Default
		Compression:   stream.Codec.FourCC,
		SizeImage:     0,
		XPelsPerMeter: 0,
		YPelsPerMeter: 0,
		ClrUsed:       0,
		ClrImportant:  0,
	}

	// Write chunk header
	chunkHeader := ChunkHeader{
		ID:   StringToChunkID(STRFChunk),
		Size: 40, // sizeof(BitmapInfoHeader)
	}

	if err := binary.Write(w.w, binary.LittleEndian, &chunkHeader); err != nil {
		return &AVIError{Op: "write strf header", Err: err}
	}

	if err := binary.Write(w.w, binary.LittleEndian, &bih); err != nil {
		return &AVIError{Op: "write bitmap info", Err: err}
	}

	return nil
}

// writeAudioFormat writes audio format info
func (w *Writer) writeAudioFormat(streamIndex int) error {
	stream := w.streams[streamIndex]

	wfx := WaveFormatEx{
		FormatTag:      1, // PCM
		Channels:       uint16(stream.Codec.Channels),
		SamplesPerSec:  uint32(stream.Codec.SampleRate),
		AvgBytesPerSec: uint32(stream.Codec.SampleRate * stream.Codec.Channels * stream.Codec.BitDepth / 8),
		BlockAlign:     uint16(stream.Codec.Channels * stream.Codec.BitDepth / 8),
		BitsPerSample:  uint16(stream.Codec.BitDepth),
		Size:           0,
	}

	// Write chunk header
	chunkHeader := ChunkHeader{
		ID:   StringToChunkID(STRFChunk),
		Size: 16, // sizeof(WaveFormatEx) without extra data
	}

	if err := binary.Write(w.w, binary.LittleEndian, &chunkHeader); err != nil {
		return &AVIError{Op: "write strf header", Err: err}
	}

	if err := binary.Write(w.w, binary.LittleEndian, &wfx); err != nil {
		return &AVIError{Op: "write wave format", Err: err}
	}

	return nil
}

// writeMOVIList writes the movie data list
func (w *Writer) writeMOVIList() error {
	moviSize := w.calculateMOVISize()

	// Write LIST header
	listHeader := LISTHeader{
		ChunkHeader: ChunkHeader{
			ID:   StringToChunkID(LISTSignature),
			Size: moviSize,
		},
		Type: StringToChunkID(MOVIList),
	}

	if err := binary.Write(w.w, binary.LittleEndian, &listHeader); err != nil {
		return &AVIError{Op: "write movi list", Err: err}
	}

	// Write packets
	for _, packet := range w.packets {
		if err := w.writePacketData(packet); err != nil {
			return err
		}
	}

	return nil
}

// writePacketData writes a single packet
func (w *Writer) writePacketData(packet Packet) error {
	// Create chunk ID (e.g., "00dc" for video, "01wb" for audio)
	var twoCC string
	if w.streams[packet.StreamIndex].Type == StreamTypeVideo {
		twoCC = "dc" // compressed video
		if packet.Flags == "K__" {
			twoCC = "db" // uncompressed video
		}
	} else if w.streams[packet.StreamIndex].Type == StreamTypeAudio {
		twoCC = "wb" // audio
	}

	chunkID := MakeChunkID(packet.StreamIndex, twoCC)

	// Write chunk header
	chunkHeader := ChunkHeader{
		ID:   chunkID,
		Size: uint32(len(packet.Data)),
	}

	if err := binary.Write(w.w, binary.LittleEndian, &chunkHeader); err != nil {
		return &AVIError{Op: "write packet header", Err: err}
	}

	// Write packet data
	if _, err := w.w.Write(packet.Data); err != nil {
		return &AVIError{Op: "write packet data", Err: err}
	}

	// Pad to even boundary
	if len(packet.Data)%2 == 1 {
		if _, err := w.w.Write([]byte{0}); err != nil {
			return &AVIError{Op: "write padding", Err: err}
		}
	}

	return nil
}

// writeIDX1Chunk writes the index chunk
func (w *Writer) writeIDX1Chunk() error {
	indexSize := len(w.packets) * 16 // sizeof(IndexEntry)

	// Write chunk header
	chunkHeader := ChunkHeader{
		ID:   StringToChunkID(IDX1Chunk),
		Size: uint32(indexSize),
	}

	if err := binary.Write(w.w, binary.LittleEndian, &chunkHeader); err != nil {
		return &AVIError{Op: "write idx1 header", Err: err}
	}

	var currentOffset uint32 = 4 // Skip movi signature
	for _, packet := range w.packets {
		var twoCC string
		if w.streams[packet.StreamIndex].Type == StreamTypeVideo {
			twoCC = "dc"
			if packet.Flags == "K__" {
				twoCC = "db"
			}
		} else if w.streams[packet.StreamIndex].Type == StreamTypeAudio {
			twoCC = "wb"
		}

		chunkID := MakeChunkID(packet.StreamIndex, twoCC)

		var flags uint32 = 0
		if packet.Flags == "K__" {
			flags = 0x10 // AVIIF_KEYFRAME
		}

		entry := IndexEntry{
			ChunkID: chunkID,
			Flags:   flags,
			Offset:  currentOffset,
			Size:    uint32(len(packet.Data)),
		}

		if err := binary.Write(w.w, binary.LittleEndian, &entry); err != nil {
			return &AVIError{Op: "write index entry", Err: err}
		}

		// Update offset for next entry
		currentOffset += 8 + AlignSize(uint32(len(packet.Data))) // chunk header + aligned data
	}

	return nil
}

// Helper functions to calculate sizes
func (w *Writer) calculateHDRLSize() uint32 {
	size := uint32(4) // hdrl signature
	size += 8 + 56    // avih chunk header + data

	for i := range w.streams {
		size += w.calculateSTRLSize(i) + 8 // strl size + LIST header
	}

	return size
}

func (w *Writer) calculateSTRLSize(streamIndex int) uint32 {
	size := uint32(4) // strl signature
	size += 8 + 56    // strh chunk header + data

	// strf chunk
	stream := w.streams[streamIndex]
	if stream.Type == StreamTypeVideo {
		size += 8 + 40 // strf header + BitmapInfoHeader
	} else if stream.Type == StreamTypeAudio {
		size += 8 + 16 // strf header + WaveFormatEx (no extra data)
	}

	return size
}

func (w *Writer) calculateMOVISize() uint32 {
	size := uint32(4) // movi signature

	for _, packet := range w.packets {
		size += 8 + AlignSize(uint32(len(packet.Data))) // chunk header + aligned data
	}

	return size
}

func (w *Writer) calculateIDX1Size() uint32 {
	return uint32(len(w.packets) * 16) // sizeof(IndexEntry)
}

// Close closes the file
func (w *Writer) Close() error {
	if w.w != nil {
		if closer, ok := w.w.(io.Closer); ok {
			return closer.Close()
		}
	}
	return nil
}