package avi

import (
	"bytes"
	"io"
	"os"
	"testing"
)

func TestSeekableBuffer(t *testing.T) {
	sb := NewSeekableBuffer()

	// Test Write
	data1 := []byte("Hello ")
	n, err := sb.Write(data1)
	if err != nil {
		t.Errorf("Write failed: %v", err)
	}
	if n != len(data1) {
		t.Errorf("Write returned %d, expected %d", n, len(data1))
	}

	// Test Seek to end and write more
	pos, err := sb.Seek(0, io.SeekEnd)
	if err != nil {
		t.Errorf("Seek failed: %v", err)
	}
	if pos != int64(len(data1)) {
		t.Errorf("Seek returned %d, expected %d", pos, len(data1))
	}

	data2 := []byte("World!")
	n, err = sb.Write(data2)
	if err != nil {
		t.Errorf("Write failed: %v", err)
	}

	// Check final content
	expected := "Hello World!"
	if string(sb.Bytes()) != expected {
		t.Errorf("Buffer contains %q, expected %q", string(sb.Bytes()), expected)
	}

	// Test Seek to middle and overwrite
	pos, err = sb.Seek(6, io.SeekStart)
	if err != nil {
		t.Errorf("Seek failed: %v", err)
	}

	data3 := []byte("Go")
	n, err = sb.Write(data3)
	if err != nil {
		t.Errorf("Write failed: %v", err)
	}

	expected = "Hello Gorld!"
	if string(sb.Bytes()) != expected {
		t.Errorf("Buffer contains %q, expected %q", string(sb.Bytes()), expected)
	}
}

func TestDemuxerWithReader(t *testing.T) {
	// Skip if test file doesn't exist
	data, err := os.ReadFile("../data/video.avi")
	if err != nil {
		t.Skipf("Test file not found: %v", err)
		return
	}

	// Create reader from bytes
	reader := bytes.NewReader(data)

	// Create demuxer
	demuxer := NewDemuxer()
	defer demuxer.Close()

	// Open from reader
	err = demuxer.Open(reader, int64(len(data)))
	if err != nil {
		t.Errorf("Failed to open from reader: %v", err)
		return
	}

	// Get file info
	fileInfo, err := demuxer.GetFileInfo()
	if err != nil {
		t.Errorf("Failed to get file info: %v", err)
		return
	}

	if fileInfo.FileSize != int64(len(data)) {
		t.Errorf("File size mismatch: got %d, expected %d", fileInfo.FileSize, len(data))
	}

	// Get streams
	streams, err := demuxer.GetStreams()
	if err != nil {
		t.Errorf("Failed to get streams: %v", err)
		return
	}

	if len(streams) == 0 {
		t.Error("No streams found")
	}

	t.Logf("Successfully opened AVI from reader: %d bytes, %d streams", len(data), len(streams))
}

func TestMuxerWithWriter(t *testing.T) {
	// Create seekable buffer
	buffer := NewSeekableBuffer()

	// Create muxer
	muxer := NewMuxer()
	defer muxer.Close()

	// Create in buffer
	err := muxer.Create(buffer)
	if err != nil {
		t.Errorf("Failed to create in buffer: %v", err)
		return
	}

	// Add video stream
	videoCodec := Codec{
		Name:   "TEST",
		FourCC: [4]byte{'T', 'E', 'S', 'T'},
		Type:   StreamTypeVideo,
		Width:  320,
		Height: 240,
		FPS:    10.0,
	}

	streamIndex, err := muxer.AddStream(videoCodec)
	if err != nil {
		t.Errorf("Failed to add stream: %v", err)
		return
	}

	// Write a packet
	packet := &Packet{
		StreamIndex: streamIndex,
		Codec:       StreamTypeVideo,
		Data:        make([]byte, 100),
		PTS:         0,
		DTS:         0,
		Duration:    1,
		Flags:       "K__",
	}

	err = muxer.WritePacket(packet)
	if err != nil {
		t.Errorf("Failed to write packet: %v", err)
		return
	}

	// Finalize
	err = muxer.Finalize()
	if err != nil {
		t.Errorf("Failed to finalize: %v", err)
		return
	}

	// Check buffer has content
	if buffer.Len() == 0 {
		t.Error("Buffer is empty after finalization")
		return
	}

	t.Logf("Successfully created AVI in buffer: %d bytes", buffer.Len())

	// Try to read it back
	reader := bytes.NewReader(buffer.Bytes())
	demuxer := NewDemuxer()
	defer demuxer.Close()

	err = demuxer.Open(reader, int64(buffer.Len()))
	if err != nil {
		t.Errorf("Failed to open created AVI from buffer: %v", err)
		return
	}

	streams, err := demuxer.GetStreams()
	if err != nil {
		t.Errorf("Failed to get streams from created AVI: %v", err)
		return
	}

	if len(streams) != 1 {
		t.Errorf("Expected 1 stream, got %d", len(streams))
	}
}