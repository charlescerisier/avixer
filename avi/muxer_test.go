package avi

import (
	"os"
	"testing"
	"time"
)

func TestNewMuxer(t *testing.T) {
	muxer := NewMuxer()
	if muxer == nil {
		t.Error("NewMuxer() returned nil")
	}
}

func TestMuxerCreate(t *testing.T) {
	muxer := NewMuxer()
	defer muxer.Close()

	// Test creating file in non-existent directory
	err := muxer.CreateFile("/nonexistent/path/test.avi")
	if err == nil {
		t.Error("Expected error when creating file in non-existent directory")
	}

	// Test creating file in current directory
	tempFile := "test_output.avi"
	defer os.Remove(tempFile)

	err = muxer.CreateFile(tempFile)
	if err != nil {
		t.Errorf("Failed to create file: %v", err)
	}

	// Test adding stream without creating file first
	muxer2 := NewMuxer()
	defer muxer2.Close()

	videoCodec := Codec{
		Name:   "TEST",
		Type:   StreamTypeVideo,
		Width:  640,
		Height: 480,
		FPS:    30.0,
	}

	_, err = muxer2.AddStream(videoCodec)
	if err == nil {
		t.Error("Expected error when adding stream without creating file")
	}
}

func TestMuxerAddStream(t *testing.T) {
	muxer := NewMuxer()
	defer muxer.Close()

	tempFile := "test_streams.avi"
	defer os.Remove(tempFile)

	err := muxer.CreateFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Add video stream
	videoCodec := Codec{
		Name:   "MJPG",
		FourCC: [4]byte{'M', 'J', 'P', 'G'},
		Type:   StreamTypeVideo,
		Width:  640,
		Height: 480,
		FPS:    30.0,
	}

	videoIndex, err := muxer.AddStream(videoCodec)
	if err != nil {
		t.Errorf("Failed to add video stream: %v", err)
	}

	if videoIndex != 0 {
		t.Errorf("Expected video stream index 0, got %d", videoIndex)
	}

	// Add audio stream
	audioCodec := Codec{
		Name:       "PCM",
		Type:       StreamTypeAudio,
		Channels:   2,
		SampleRate: 44100,
		BitDepth:   16,
	}

	audioIndex, err := muxer.AddStream(audioCodec)
	if err != nil {
		t.Errorf("Failed to add audio stream: %v", err)
	}

	if audioIndex != 1 {
		t.Errorf("Expected audio stream index 1, got %d", audioIndex)
	}
}

func TestMuxerWritePacket(t *testing.T) {
	muxer := NewMuxer()
	defer muxer.Close()

	tempFile := "test_packets.avi"
	defer os.Remove(tempFile)

	err := muxer.CreateFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Add video stream
	videoCodec := Codec{
		Name:   "TEST",
		Type:   StreamTypeVideo,
		Width:  320,
		Height: 240,
		FPS:    15.0,
	}

	streamIndex, err := muxer.AddStream(videoCodec)
	if err != nil {
		t.Fatalf("Failed to add stream: %v", err)
	}

	// Test writing packet with invalid stream index
	invalidPacket := &Packet{
		StreamIndex: 99,
		Codec:       StreamTypeVideo,
		Data:        []byte{1, 2, 3, 4},
		PTS:         0,
		DTS:         0,
		Duration:    1,
		Flags:       "K__",
	}

	err = muxer.WritePacket(invalidPacket)
	if err == nil {
		t.Error("Expected error when writing packet with invalid stream index")
	}

	// Test writing valid packet
	validPacket := &Packet{
		StreamIndex: streamIndex,
		Codec:       StreamTypeVideo,
		Data:        []byte{1, 2, 3, 4, 5, 6, 7, 8},
		PTS:         0,
		DTS:         0,
		Duration:    1,
		Flags:       "K__",
	}

	err = muxer.WritePacket(validPacket)
	if err != nil {
		t.Errorf("Failed to write valid packet: %v", err)
	}
}

func TestMuxerFinalize(t *testing.T) {
	muxer := NewMuxer()
	defer muxer.Close()

	// Test finalizing without creating file
	err := muxer.Finalize()
	if err == nil {
		t.Error("Expected error when finalizing without creating file")
	}

	tempFile := "test_finalize.avi"
	defer os.Remove(tempFile)

	err = muxer.CreateFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Add a stream and packet
	codec := Codec{
		Name:   "TEST",
		Type:   StreamTypeVideo,
		Width:  160,
		Height: 120,
		FPS:    10.0,
	}

	streamIndex, err := muxer.AddStream(codec)
	if err != nil {
		t.Fatalf("Failed to add stream: %v", err)
	}

	packet := &Packet{
		StreamIndex: streamIndex,
		Codec:       StreamTypeVideo,
		Data:        make([]byte, 100), // 100 bytes of data
		PTS:         0,
		DTS:         0,
		Duration:    1,
		Flags:       "K__",
	}

	err = muxer.WritePacket(packet)
	if err != nil {
		t.Fatalf("Failed to write packet: %v", err)
	}

	// Finalize the file
	err = muxer.Finalize()
	if err != nil {
		t.Errorf("Failed to finalize file: %v", err)
	}

	// Check that file exists and has content
	info, err := os.Stat(tempFile)
	if err != nil {
		t.Errorf("File not found after finalize: %v", err)
	}

	if info.Size() == 0 {
		t.Error("File is empty after finalize")
	}

	t.Logf("Created AVI file: %s (size: %d bytes)", tempFile, info.Size())
}

func TestMuxerFullWorkflow(t *testing.T) {
	muxer := NewMuxer()
	defer muxer.Close()

	tempFile := "test_full_workflow.avi"
	defer os.Remove(tempFile)

	// Create file
	err := muxer.CreateFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Add video stream
	videoCodec := Codec{
		Name:   "MJPG",
		FourCC: [4]byte{'M', 'J', 'P', 'G'},
		Type:   StreamTypeVideo,
		Width:  320,
		Height: 240,
		FPS:    25.0,
	}

	videoIndex, err := muxer.AddStream(videoCodec)
	if err != nil {
		t.Fatalf("Failed to add video stream: %v", err)
	}

	// Add audio stream
	audioCodec := Codec{
		Name:       "PCM",
		Type:       StreamTypeAudio,
		Channels:   1,
		SampleRate: 22050,
		BitDepth:   16,
	}

	audioIndex, err := muxer.AddStream(audioCodec)
	if err != nil {
		t.Fatalf("Failed to add audio stream: %v", err)
	}

	// Write some packets
	for i := 0; i < 5; i++ {
		// Video packet
		videoPacket := &Packet{
			StreamIndex:  videoIndex,
			Codec:        StreamTypeVideo,
			Data:         make([]byte, 1000+i*100), // Variable size
			PTS:          int64(i),
			DTS:          int64(i),
			Duration:     1,
			PTSTime:      time.Duration(i) * time.Second / 25,
			DTSTime:      time.Duration(i) * time.Second / 25,
			DurationTime: time.Second / 25,
			Flags:        "K__",
		}

		if i > 0 {
			videoPacket.Flags = "___" // Non-keyframe
		}

		err = muxer.WritePacket(videoPacket)
		if err != nil {
			t.Fatalf("Failed to write video packet %d: %v", i, err)
		}

		// Audio packet
		audioPacket := &Packet{
			StreamIndex:  audioIndex,
			Codec:        StreamTypeAudio,
			Data:         make([]byte, 1024), // Fixed size
			PTS:          int64(i * 1024),
			DTS:          int64(i * 1024),
			Duration:     1024,
			PTSTime:      time.Duration(i*1024) * time.Second / 22050,
			DTSTime:      time.Duration(i*1024) * time.Second / 22050,
			DurationTime: time.Duration(1024) * time.Second / 22050,
			Flags:        "K__",
		}

		err = muxer.WritePacket(audioPacket)
		if err != nil {
			t.Fatalf("Failed to write audio packet %d: %v", i, err)
		}
	}

	// Finalize
	err = muxer.Finalize()
	if err != nil {
		t.Fatalf("Failed to finalize: %v", err)
	}

	// Verify file size
	info, err := os.Stat(tempFile)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	if info.Size() < 1000 {
		t.Errorf("File seems too small: %d bytes", info.Size())
	}

	t.Logf("Successfully created AVI file with %d bytes", info.Size())

	// Try to read it back with demuxer
	demuxer := NewDemuxer()
	defer demuxer.Close()

	err = demuxer.OpenFile(tempFile)
	if err != nil {
		t.Errorf("Failed to open created file with demuxer: %v", err)
		return
	}

	fileInfo, err := demuxer.GetFileInfo()
	if err != nil {
		t.Errorf("Failed to get file info: %v", err)
		return
	}

	if fileInfo.VideoStreams != 1 {
		t.Errorf("Expected 1 video stream, got %d", fileInfo.VideoStreams)
	}

	if fileInfo.AudioStreams != 1 {
		t.Errorf("Expected 1 audio stream, got %d", fileInfo.AudioStreams)
	}

	streams, err := demuxer.GetStreams()
	if err != nil {
		t.Errorf("Failed to get streams: %v", err)
		return
	}

	if len(streams) != 2 {
		t.Errorf("Expected 2 streams, got %d", len(streams))
	}

	t.Logf("Successfully verified created AVI file")
}