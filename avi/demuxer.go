package avi

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"time"
)

// NewDemuxer creates a new AVI demuxer
func NewDemuxer() Demuxer {
	return &Reader{}
}

// Open opens an AVI reader
func (r *Reader) Open(reader io.ReadSeeker, size int64) error {
	r.r = reader
	r.fileSize = size
	r.filename = "" // No filename when using reader directly

	// Parse the file structure
	if err := r.parseFile(); err != nil {
		return err
	}

	return nil
}

// OpenFile opens an AVI file for reading (convenience method)
func (r *Reader) OpenFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return &AVIError{Op: "open", Err: err}
	}

	// Get file size
	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return &AVIError{Op: "stat", Err: err}
	}

	r.filename = filename
	
	// Use the generic Open method
	if err := r.Open(file, stat.Size()); err != nil {
		file.Close()
		return err
	}

	return nil
}

// parseFile parses the AVI file structure
func (r *Reader) parseFile() error {
	// Read RIFF header
	var riffHeader RIFFHeader
	if err := binary.Read(r.r, binary.LittleEndian, &riffHeader); err != nil {
		return &AVIError{Op: "read riff header", Err: err}
	}

	if !IsValidRIFFSignature(riffHeader.Signature) {
		return &AVIError{Op: "validate riff", Err: fmt.Errorf("not a RIFF file")}
	}

	if !IsValidAVISignature(riffHeader.Type) {
		return &AVIError{Op: "validate avi", Err: fmt.Errorf("not an AVI file")}
	}

	// Parse the file structure
	fileSize := int64(riffHeader.FileSize + 8)
	if fileSize != r.fileSize {
		// Some files have incorrect size in header, continue anyway
	}

	return r.parseChunks()
}

// parseChunks parses all chunks in the file
func (r *Reader) parseChunks() error {
	var streams []Stream
	var fileInfo FileInfo
	
	fileInfo.Filename = r.filename
	fileInfo.FileSize = r.fileSize

	// Skip to after RIFF header
	if _, err := r.r.Seek(12, io.SeekStart); err != nil {
		return &AVIError{Op: "seek", Err: err}
	}

	for {
		pos, err := r.r.Seek(0, io.SeekCurrent)
		if err != nil {
			break
		}
		
		if pos >= r.fileSize-8 {
			break
		}

		var header ChunkHeader
		if err := binary.Read(r.r, binary.LittleEndian, &header); err != nil {
			if err == io.EOF {
				break
			}
			return &AVIError{Op: "read chunk header", Err: err}
		}

		chunkID := ChunkIDToString(header.ID)
		
		switch chunkID {
		case LISTSignature:
			if err := r.parseLISTChunk(header.Size, &streams, &fileInfo); err != nil {
				return err
			}
		case IDX1Chunk:
			// Parse index for packet reading
			if err := r.parseIDX1Chunk(header.Size); err != nil {
				return err
			}
		default:
			// Skip unknown chunks
			if _, err := r.r.Seek(int64(AlignSize(header.Size)), io.SeekCurrent); err != nil {
				return &AVIError{Op: "skip chunk", Err: err}
			}
		}
	}

	r.streams = streams
	r.fileInfo = &fileInfo
	r.fileInfo.Streams = streams
	
	// Count stream types
	for _, stream := range streams {
		switch stream.Type {
		case StreamTypeVideo:
			r.fileInfo.VideoStreams++
		case StreamTypeAudio:
			r.fileInfo.AudioStreams++
		}
	}

	return nil
}

// parseLISTChunk parses a LIST chunk
func (r *Reader) parseLISTChunk(size uint32, streams *[]Stream, fileInfo *FileInfo) error {
	var listType [4]byte
	if err := binary.Read(r.r, binary.LittleEndian, &listType); err != nil {
		return &AVIError{Op: "read list type", Err: err}
	}

	listTypeStr := string(listType[:])
	remainingSize := size - 4

	switch listTypeStr {
	case HDRLList:
		return r.parseHDRLList(remainingSize, streams, fileInfo)
	case MOVIList:
		// Store movi offset for packet reading
		// Current position is after reading "movi" signature, so we need to subtract 4
		currentPos, _ := r.r.Seek(0, io.SeekCurrent)
		r.moviOffset = currentPos - 4 // Subtract the "movi" signature we just read
		// Skip movi list data for now
		if _, err := r.r.Seek(int64(AlignSize(remainingSize)), io.SeekCurrent); err != nil {
			return &AVIError{Op: "skip movi", Err: err}
		}
	default:
		// Skip unknown list
		if _, err := r.r.Seek(int64(AlignSize(remainingSize)), io.SeekCurrent); err != nil {
			return &AVIError{Op: "skip list", Err: err}
		}
	}

	return nil
}

// parseHDRLList parses the header list
func (r *Reader) parseHDRLList(size uint32, streams *[]Stream, fileInfo *FileInfo) error {
	endPos, err := r.r.Seek(0, io.SeekCurrent)
	if err != nil {
		return &AVIError{Op: "get position", Err: err}
	}
	endPos += int64(size)

	for {
		pos, err := r.r.Seek(0, io.SeekCurrent)
		if err != nil || pos >= endPos {
			break
		}

		var header ChunkHeader
		if err := binary.Read(r.r, binary.LittleEndian, &header); err != nil {
			if err == io.EOF {
				break
			}
			return &AVIError{Op: "read hdrl chunk", Err: err}
		}

		chunkID := ChunkIDToString(header.ID)

		switch chunkID {
		case AVIHChunk:
			if err := r.parseAVIHChunk(header.Size, fileInfo); err != nil {
				return err
			}
		case LISTSignature:
			if err := r.parseSTRLList(header.Size, streams); err != nil {
				return err
			}
		default:
			// Skip unknown chunk
			if _, err := r.r.Seek(int64(AlignSize(header.Size)), io.SeekCurrent); err != nil {
				return &AVIError{Op: "skip hdrl chunk", Err: err}
			}
		}
	}

	return nil
}

// parseAVIHChunk parses the main AVI header
func (r *Reader) parseAVIHChunk(size uint32, fileInfo *FileInfo) error {
	var header AVIMainHeader
	if err := binary.Read(r.r, binary.LittleEndian, &header); err != nil {
		return &AVIError{Op: "read avih", Err: err}
	}

	if header.MicroSecPerFrame > 0 {
		fileInfo.Duration = time.Duration(header.TotalFrames) * time.Duration(header.MicroSecPerFrame) * time.Microsecond
	}

	// Skip any remaining bytes in chunk
	if size > 56 { // sizeof(AVIMainHeader)
		if _, err := r.r.Seek(int64(size-56), io.SeekCurrent); err != nil {
			return &AVIError{Op: "skip avih remainder", Err: err}
		}
	}

	return nil
}

// parseSTRLList parses a stream list
func (r *Reader) parseSTRLList(size uint32, streams *[]Stream) error {
	var listType [4]byte
	if err := binary.Read(r.r, binary.LittleEndian, &listType); err != nil {
		return &AVIError{Op: "read strl type", Err: err}
	}

	if string(listType[:]) != STRLList {
		// Skip if not a stream list
		if _, err := r.r.Seek(int64(AlignSize(size-4)), io.SeekCurrent); err != nil {
			return &AVIError{Op: "skip non-strl", Err: err}
		}
		return nil
	}

	var stream Stream
	stream.Index = len(*streams)

	endPos, err := r.r.Seek(0, io.SeekCurrent)
	if err != nil {
		return &AVIError{Op: "get strl position", Err: err}
	}
	endPos += int64(size - 4)

	for {
		pos, err := r.r.Seek(0, io.SeekCurrent)
		if err != nil || pos >= endPos {
			break
		}

		var header ChunkHeader
		if err := binary.Read(r.r, binary.LittleEndian, &header); err != nil {
			if err == io.EOF {
				break
			}
			return &AVIError{Op: "read strl chunk", Err: err}
		}

		chunkID := ChunkIDToString(header.ID)

		switch chunkID {
		case STRHChunk:
			if err := r.parseSTRHChunk(header.Size, &stream); err != nil {
				return err
			}
		case STRFChunk:
			if err := r.parseSTRFChunk(header.Size, &stream); err != nil {
				return err
			}
		default:
			// Skip unknown chunk (strn, strd, etc.)
			if _, err := r.r.Seek(int64(AlignSize(header.Size)), io.SeekCurrent); err != nil {
				return &AVIError{Op: "skip strl chunk", Err: err}
			}
		}
	}

	*streams = append(*streams, stream)
	return nil
}

// parseSTRHChunk parses a stream header
func (r *Reader) parseSTRHChunk(size uint32, stream *Stream) error {
	var header AVIStreamHeader
	if err := binary.Read(r.r, binary.LittleEndian, &header); err != nil {
		return &AVIError{Op: "read strh", Err: err}
	}

	// Determine stream type
	if IsVideoStream(header.Type) {
		stream.Type = StreamTypeVideo
		stream.Codec.Type = StreamTypeVideo
	} else if IsAudioStream(header.Type) {
		stream.Type = StreamTypeAudio  
		stream.Codec.Type = StreamTypeAudio
	}

	// Set codec handler
	stream.Codec.FourCC = header.Handler
	
	// Clean up codec name (remove null bytes and unprintable characters)
	codecName := string(header.Handler[:])
	cleanName := ""
	for _, b := range []byte(codecName) {
		if b >= 32 && b <= 126 { // Printable ASCII
			cleanName += string(b)
		}
	}
	stream.Codec.Name = cleanName

	// Calculate duration and frame rate
	if header.Rate > 0 && header.Scale > 0 {
		if stream.Type == StreamTypeVideo {
			stream.Codec.FPS = float64(header.Rate) / float64(header.Scale)
		}
		if header.Length > 0 {
			stream.Duration = time.Duration(header.Length) * time.Duration(header.Scale) * time.Second / time.Duration(header.Rate)
		}
	}

	// Skip any remaining bytes
	if size > 56 { // sizeof(AVIStreamHeader)  
		if _, err := r.r.Seek(int64(size-56), io.SeekCurrent); err != nil {
			return &AVIError{Op: "skip strh remainder", Err: err}
		}
	}

	return nil
}

// parseSTRFChunk parses stream format chunk
func (r *Reader) parseSTRFChunk(size uint32, stream *Stream) error {
	if stream.Type == StreamTypeVideo {
		return r.parseVideoFormat(size, stream)
	} else if stream.Type == StreamTypeAudio {
		return r.parseAudioFormat(size, stream)
	} else {
		// Skip unknown format
		if _, err := r.r.Seek(int64(AlignSize(size)), io.SeekCurrent); err != nil {
			return &AVIError{Op: "skip strf", Err: err}
		}
	}
	return nil
}

// parseVideoFormat parses video format info
func (r *Reader) parseVideoFormat(size uint32, stream *Stream) error {
	var bih BitmapInfoHeader
	if err := binary.Read(r.r, binary.LittleEndian, &bih); err != nil {
		return &AVIError{Op: "read bitmap info", Err: err}
	}

	stream.Codec.Width = int(bih.Width)
	stream.Codec.Height = int(bih.Height)
	if bih.Height < 0 {
		stream.Codec.Height = -stream.Codec.Height
	}

	// Skip remaining bytes
	if size > 40 { // sizeof(BitmapInfoHeader)
		if _, err := r.r.Seek(int64(size-40), io.SeekCurrent); err != nil {
			return &AVIError{Op: "skip bitmap remainder", Err: err}
		}
	}

	return nil
}

// parseAudioFormat parses audio format info  
func (r *Reader) parseAudioFormat(size uint32, stream *Stream) error {
	var wfx WaveFormatEx
	if err := binary.Read(r.r, binary.LittleEndian, &wfx); err != nil {
		return &AVIError{Op: "read wave format", Err: err}
	}

	stream.Codec.Channels = int(wfx.Channels)
	stream.Codec.SampleRate = int(wfx.SamplesPerSec)
	stream.Codec.BitDepth = int(wfx.BitsPerSample)

	// Skip remaining bytes
	if size > 18 { // sizeof(WaveFormatEx) without extra data
		if _, err := r.r.Seek(int64(size-18), io.SeekCurrent); err != nil {
			return &AVIError{Op: "skip wave remainder", Err: err}
		}
	}

	return nil
}

// GetFileInfo returns metadata about the file
func (r *Reader) GetFileInfo() (*FileInfo, error) {
	if r.fileInfo == nil {
		return nil, &AVIError{Op: "get file info", Err: fmt.Errorf("file not opened")}
	}
	return r.fileInfo, nil
}

// GetStreams returns all streams in the file
func (r *Reader) GetStreams() ([]Stream, error) {
	if r.streams == nil {
		return nil, &AVIError{Op: "get streams", Err: fmt.Errorf("file not opened")}
	}
	return r.streams, nil
}

// ReadPacket reads the next packet from the file
func (r *Reader) ReadPacket() (*Packet, error) {
	// This is a simplified implementation
	// In practice, you'd seek to the movi chunk and read packets sequentially
	return nil, &AVIError{Op: "read packet", Err: fmt.Errorf("not implemented yet")}
}

// ReadPacketData reads the actual data for a packet at the given position
func (r *Reader) ReadPacketData(packet *Packet) ([]byte, error) {
	if r.r == nil {
		return nil, &AVIError{Op: "read packet data", Err: fmt.Errorf("file not open")}
	}

	// Save current position
	currentPos, err := r.r.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, &AVIError{Op: "get current position", Err: err}
	}

	// Seek to packet position (which points to chunk header)
	if _, err := r.r.Seek(packet.Position, io.SeekStart); err != nil {
		return nil, &AVIError{Op: "seek to packet", Err: err}
	}

	// Read chunk header to verify and get actual size
	var header ChunkHeader
	if err := binary.Read(r.r, binary.LittleEndian, &header); err != nil {
		return nil, &AVIError{Op: "read packet header", Err: err}
	}


	// Use the size from the chunk header (actual file size)
	dataSize := header.Size
	
	// Get current position to check if we have enough data left
	currentPosAfterHeader, err := r.r.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, &AVIError{Op: "get position after header", Err: err}
	}
	
	// Debug: check if we're trying to read beyond file bounds
	if currentPosAfterHeader + int64(dataSize) > r.fileSize {
		return nil, &AVIError{Op: "read packet data", Err: fmt.Errorf("packet would read beyond file bounds: pos=%d, size=%d, filesize=%d", currentPosAfterHeader, dataSize, r.fileSize)}
	}
	
	// Read packet data
	data := make([]byte, dataSize)
	if _, err := io.ReadFull(r.r, data); err != nil {
		return nil, &AVIError{Op: "read packet data", Err: err}
	}

	// Restore position
	if _, err := r.r.Seek(currentPos, io.SeekStart); err != nil {
		return nil, &AVIError{Op: "restore position", Err: err}
	}

	return data, nil
}

// Seek seeks to a specific timestamp
func (r *Reader) Seek(timestamp time.Duration) error {
	// This would require index parsing
	return &AVIError{Op: "seek", Err: fmt.Errorf("not implemented yet")}
}

// parseIDX1Chunk parses the index chunk
func (r *Reader) parseIDX1Chunk(size uint32) error {
	numEntries := size / 16 // sizeof(IndexEntry)
	r.indexEntries = make([]IndexEntry, numEntries)
	
	for i := uint32(0); i < numEntries; i++ {
		if err := binary.Read(r.r, binary.LittleEndian, &r.indexEntries[i]); err != nil {
			return &AVIError{Op: "read index entry", Err: err}
		}
	}
	
	return nil
}

// ReadAllPackets reads all packets from the file
func (r *Reader) ReadAllPackets() ([]Packet, error) {
	if len(r.indexEntries) == 0 {
		return nil, &AVIError{Op: "read packets", Err: fmt.Errorf("no index entries found")}
	}
	
	var packets []Packet
	
	for _, entry := range r.indexEntries {
		// Parse stream index and type from chunk ID
		chunkID := ChunkIDToString(entry.ChunkID)
		if len(chunkID) < 4 {
			continue
		}
		
		// Extract stream index (first 2 digits)
		streamIndex := int((chunkID[0]-'0')*10 + (chunkID[1]-'0'))
		if streamIndex >= len(r.streams) {
			continue
		}
		
		// Extract chunk type (last 2 chars)
		chunkType := chunkID[2:4]
		
		// Determine codec type
		var codecType StreamType
		switch chunkType {
		case "dc", "db": // video chunks
			codecType = StreamTypeVideo
		case "wb": // audio chunks
			codecType = StreamTypeAudio
		default:
			continue
		}
		
		// Calculate timestamp based on stream type and properties
		var pts, dts int64
		var ptsTime, dtsTime, durationTime time.Duration
		
		if codecType == StreamTypeVideo {
			// Count video frames for this stream
			videoFrameCount := int64(0)
			for _, p := range packets {
				if p.StreamIndex == streamIndex && p.Codec == StreamTypeVideo {
					videoFrameCount++
				}
			}
			dts = videoFrameCount
			pts = dts
			
			if r.streams[streamIndex].Codec.FPS > 0 {
				frameDuration := time.Second / time.Duration(r.streams[streamIndex].Codec.FPS)
				dtsTime = time.Duration(dts) * frameDuration
				ptsTime = dtsTime
				durationTime = frameDuration
			}
		} else if codecType == StreamTypeAudio {
			// Count audio packets for this stream
			audioPacketCount := int64(0)
			for _, p := range packets {
				if p.StreamIndex == streamIndex && p.Codec == StreamTypeAudio {
					audioPacketCount++
				}
			}
			// Each audio packet is typically 1024 samples
			samplesPerPacket := int64(1024)
			dts = audioPacketCount * samplesPerPacket
			pts = dts // For AVI, PTS equals DTS for audio
			
			if r.streams[streamIndex].Codec.SampleRate > 0 {
				sampleDuration := time.Second / time.Duration(r.streams[streamIndex].Codec.SampleRate)
				dtsTime = time.Duration(dts) * sampleDuration
				ptsTime = dtsTime
				durationTime = time.Duration(samplesPerPacket) * sampleDuration
			}
		}
		
		// Determine flags
		flags := "___"
		if entry.Flags&0x10 != 0 { // AVIIF_KEYFRAME
			flags = "K__"
		}
		
		position := int64(entry.Offset) + r.moviOffset // Remove +4 since moviOffset now points to correct position
		
		
		packet := Packet{
			StreamIndex:  streamIndex,
			Codec:        codecType,
			Data:         nil, // We don't read actual data for metadata
			PTS:          pts,
			DTS:          dts,
			Duration:     1024, // Default for audio, 1 for video
			Size:         int(entry.Size),
			Position:     position,
			Flags:        flags,
			PTSTime:      ptsTime,
			DTSTime:      dtsTime,
			DurationTime: durationTime,
		}
		
		if codecType == StreamTypeVideo {
			packet.Duration = 1
		}
		
		packets = append(packets, packet)
	}
	
	return packets, nil
}

// Close closes the file
func (r *Reader) Close() error {
	if r.r != nil {
		if closer, ok := r.r.(io.Closer); ok {
			return closer.Close()
		}
	}
	return nil
}