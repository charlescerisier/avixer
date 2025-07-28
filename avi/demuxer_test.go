package avi

import (
	"testing"
)

func TestNewDemuxer(t *testing.T) {
	demuxer := NewDemuxer()
	if demuxer == nil {
		t.Error("NewDemuxer() returned nil")
	}
}

func TestDemuxerOpen(t *testing.T) {
	demuxer := NewDemuxer()
	defer demuxer.Close()

	// Test opening non-existent file
	err := demuxer.OpenFile("nonexistent.avi")
	if err == nil {
		t.Error("Expected error when opening non-existent file")
	}

	// Test opening existing file (if video.avi exists)
	err = demuxer.OpenFile("../video.avi")
	if err != nil {
		t.Logf("Could not open test file (this is expected if ../video.avi doesn't exist): %v", err)
		return
	}

	// If we successfully opened the file, test getting file info
	fileInfo, err := demuxer.GetFileInfo()
	if err != nil {
		t.Errorf("Failed to get file info: %v", err)
		return
	}

	if fileInfo == nil {
		t.Error("GetFileInfo() returned nil")
		return
	}

	// Basic validation
	if fileInfo.FileSize <= 0 {
		t.Error("File size should be greater than 0")
	}

	if fileInfo.Filename == "" {
		t.Error("Filename should not be empty")
	}

	t.Logf("File info: size=%d, duration=%v, video_streams=%d, audio_streams=%d",
		fileInfo.FileSize, fileInfo.Duration, fileInfo.VideoStreams, fileInfo.AudioStreams)
}

func TestDemuxerGetStreams(t *testing.T) {
	demuxer := NewDemuxer()
	defer demuxer.Close()

	// Test getting streams without opening file
	_, err := demuxer.GetStreams()
	if err == nil {
		t.Error("Expected error when getting streams without opening file")
	}

	// Test with opened file
	err = demuxer.OpenFile("../video.avi")
	if err != nil {
		t.Logf("Could not open test file: %v", err)
		return
	}

	streams, err := demuxer.GetStreams()
	if err != nil {
		t.Errorf("Failed to get streams: %v", err)
		return
	}

	if len(streams) == 0 {
		t.Error("Expected at least one stream")
		return
	}

	// Validate stream info
	for i, stream := range streams {
		if stream.Index != i {
			t.Errorf("Stream %d: expected index %d, got %d", i, i, stream.Index)
		}

		if stream.Type != StreamTypeVideo && stream.Type != StreamTypeAudio {
			t.Errorf("Stream %d: invalid stream type %s", i, stream.Type)
		}

		if stream.Type == StreamTypeVideo {
			if stream.Codec.Width <= 0 || stream.Codec.Height <= 0 {
				t.Errorf("Stream %d: invalid video dimensions %dx%d", i, stream.Codec.Width, stream.Codec.Height)
			}
		}

		if stream.Type == StreamTypeAudio {
			if stream.Codec.SampleRate <= 0 {
				t.Errorf("Stream %d: invalid sample rate %d", i, stream.Codec.SampleRate)
			}
		}

		t.Logf("Stream %d: type=%s, codec=%s", i, stream.Type, stream.Codec.Name)
	}
}

func TestChunkIDHelpers(t *testing.T) {
	// Test MakeChunkID
	id := MakeChunkID(0, "dc")
	expected := [4]byte{'0', '0', 'd', 'c'}
	if id != expected {
		t.Errorf("MakeChunkID(0, \"dc\") = %v, expected %v", id, expected)
	}

	id = MakeChunkID(15, "wb")
	expected = [4]byte{'1', '5', 'w', 'b'}
	if id != expected {
		t.Errorf("MakeChunkID(15, \"wb\") = %v, expected %v", id, expected)
	}

	// Test ChunkIDToString
	str := ChunkIDToString([4]byte{'R', 'I', 'F', 'F'})
	if str != "RIFF" {
		t.Errorf("ChunkIDToString([4]byte{'R', 'I', 'F', 'F'}) = %s, expected RIFF", str)
	}

	// Test StringToChunkID
	id = StringToChunkID("LIST")
	expected = [4]byte{'L', 'I', 'S', 'T'}
	if id != expected {
		t.Errorf("StringToChunkID(\"LIST\") = %v, expected %v", id, expected)
	}
}

func TestValidationFunctions(t *testing.T) {
	// Test RIFF signature validation
	if !IsValidRIFFSignature([4]byte{'R', 'I', 'F', 'F'}) {
		t.Error("IsValidRIFFSignature should return true for RIFF")
	}

	if IsValidRIFFSignature([4]byte{'X', 'Y', 'Z', 'W'}) {
		t.Error("IsValidRIFFSignature should return false for invalid signature")
	}

	// Test AVI signature validation
	if !IsValidAVISignature([4]byte{'A', 'V', 'I', ' '}) {
		t.Error("IsValidAVISignature should return true for 'AVI '")
	}

	// Test stream type validation
	if !IsVideoStream([4]byte{'v', 'i', 'd', 's'}) {
		t.Error("IsVideoStream should return true for 'vids'")
	}

	if !IsAudioStream([4]byte{'a', 'u', 'd', 's'}) {
		t.Error("IsAudioStream should return true for 'auds'")
	}
}

func TestAVIError(t *testing.T) {
	err := &AVIError{
		Op:  "test operation",
		Err: nil,
	}

	expectedMsg := "avi: test operation: <nil>"
	if err.Error() != expectedMsg {
		t.Errorf("AVIError.Error() = %s, expected %s", err.Error(), expectedMsg)
	}
}

func TestAlignSize(t *testing.T) {
	tests := []struct {
		input    uint32
		expected uint32
	}{
		{0, 0},
		{1, 2},
		{2, 2},
		{3, 4},
		{4, 4},
		{5, 6},
		{100, 100},
		{101, 102},
	}

	for _, test := range tests {
		result := AlignSize(test.input)
		if result != test.expected {
			t.Errorf("AlignSize(%d) = %d, expected %d", test.input, result, test.expected)
		}
	}
}