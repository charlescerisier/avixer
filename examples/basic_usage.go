package main

import (
	"fmt"
	"log"

	"github.com/charlescerisier/avixer/avi"
)

func main() {
	fmt.Println("Avixer Basic Usage Example")
	fmt.Println("==========================")

	// Example 1: Reading an AVI file
	fmt.Println("\n1. Reading AVI file...")
	readExample()

	// Example 2: Creating an AVI file
	fmt.Println("\n2. Creating AVI file...")
	writeExample()

	fmt.Println("\nExample completed!")
}

func readExample() {
	// Create demuxer
	demuxer := avi.NewDemuxer()
	defer demuxer.Close()

	// Try to open the video file (adjust path as needed)
	err := demuxer.OpenFile("../video.avi")
	if err != nil {
		fmt.Printf("Could not open video file: %v\n", err)
		fmt.Println("Make sure you have a video.avi file in the parent directory")
		return
	}

	// Get file information
	fileInfo, err := demuxer.GetFileInfo()
	if err != nil {
		log.Printf("Failed to get file info: %v", err)
		return
	}

	fmt.Printf("File: %s\n", fileInfo.Filename)
	fmt.Printf("Size: %d bytes\n", fileInfo.FileSize)
	fmt.Printf("Duration: %v\n", fileInfo.Duration)
	fmt.Printf("Video streams: %d\n", fileInfo.VideoStreams)
	fmt.Printf("Audio streams: %d\n", fileInfo.AudioStreams)

	// Get stream details
	streams, err := demuxer.GetStreams()
	if err != nil {
		log.Printf("Failed to get streams: %v", err)
		return
	}

	fmt.Printf("\nStreams:\n")
	for _, stream := range streams {
		fmt.Printf("  Stream #%d: %s\n", stream.Index, stream.Type)
		fmt.Printf("    Codec: %s\n", stream.Codec.Name)
		
		if stream.Type == avi.StreamTypeVideo {
			fmt.Printf("    Resolution: %dx%d\n", stream.Codec.Width, stream.Codec.Height)
			fmt.Printf("    FPS: %.2f\n", stream.Codec.FPS)
		} else if stream.Type == avi.StreamTypeAudio {
			fmt.Printf("    Sample rate: %d Hz\n", stream.Codec.SampleRate)
			fmt.Printf("    Channels: %d\n", stream.Codec.Channels)
			fmt.Printf("    Bit depth: %d\n", stream.Codec.BitDepth)
		}
		
		if stream.Duration > 0 {
			fmt.Printf("    Duration: %v\n", stream.Duration)
		}
	}
}

func writeExample() {
	// Create muxer
	muxer := avi.NewMuxer()
	defer muxer.Close()

	// Create output file
	outputFile := "example_output.avi"
	err := muxer.CreateFile(outputFile)
	if err != nil {
		log.Printf("Failed to create file: %v", err)
		return
	}

	fmt.Printf("Creating: %s\n", outputFile)

	// Add video stream
	videoCodec := avi.Codec{
		Name:   "MJPG",
		FourCC: [4]byte{'M', 'J', 'P', 'G'},
		Type:   avi.StreamTypeVideo,
		Width:  640,
		Height: 480,
		FPS:    30.0,
	}

	videoStreamIndex, err := muxer.AddStream(videoCodec)
	if err != nil {
		log.Printf("Failed to add video stream: %v", err)
		return
	}

	fmt.Printf("Added video stream #%d: %dx%d @ %.1f fps\n", 
		videoStreamIndex, videoCodec.Width, videoCodec.Height, videoCodec.FPS)

	// Add audio stream
	audioCodec := avi.Codec{
		Name:       "PCM",
		Type:       avi.StreamTypeAudio,
		Channels:   2,
		SampleRate: 44100,
		BitDepth:   16,
	}

	audioStreamIndex, err := muxer.AddStream(audioCodec)
	if err != nil {
		log.Printf("Failed to add audio stream: %v", err)
		return
	}

	fmt.Printf("Added audio stream #%d: %d Hz, %d channels, %d bit\n",
		audioStreamIndex, audioCodec.SampleRate, audioCodec.Channels, audioCodec.BitDepth)

	// Write some sample frames
	fmt.Println("Writing sample frames...")
	for i := 0; i < 10; i++ {
		// Create dummy video frame (in real usage, this would be actual video data)
		videoData := make([]byte, 1000+i*100) // Variable size to simulate real frames
		for j := range videoData {
			videoData[j] = byte(i * 10) // Simple pattern
		}

		videoPacket := &avi.Packet{
			StreamIndex: videoStreamIndex,
			Codec:       avi.StreamTypeVideo,
			Data:        videoData,
			PTS:         int64(i),
			DTS:         int64(i),
			Duration:    1,
			Flags:       "K__", // Mark as keyframe
		}

		if i > 0 && i%5 != 0 {
			videoPacket.Flags = "___" // Non-keyframe
		}

		err = muxer.WritePacket(videoPacket)
		if err != nil {
			log.Printf("Failed to write video packet %d: %v", i, err)
			return
		}

		// Create dummy audio sample (in real usage, this would be actual audio data)
		audioData := make([]byte, 1024) // Fixed size for audio
		for j := range audioData {
			audioData[j] = byte((i + j) % 256) // Simple pattern
		}

		audioPacket := &avi.Packet{
			StreamIndex: audioStreamIndex,
			Codec:       avi.StreamTypeAudio,
			Data:        audioData,
			PTS:         int64(i * 1024),
			DTS:         int64(i * 1024),
			Duration:    1024,
			Flags:       "K__",
		}

		err = muxer.WritePacket(audioPacket)
		if err != nil {
			log.Printf("Failed to write audio packet %d: %v", i, err)
			return
		}

		fmt.Printf("  Frame %d: video=%d bytes, audio=%d bytes\n", 
			i, len(videoData), len(audioData))
	}

	// Finalize the file (writes headers and indices)
	fmt.Println("Finalizing file...")
	err = muxer.Finalize()
	if err != nil {
		log.Printf("Failed to finalize file: %v", err)
		return
	}

	fmt.Printf("Successfully created %s\n", outputFile)
	fmt.Println("You can now analyze it with: ./avixer -i example_output.avi")
}