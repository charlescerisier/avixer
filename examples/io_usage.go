// +build ignore

package main

import (
	"bytes"
	"fmt"
	"log"
	"os"

	"github.com/charlescerisier/avixer/avi"
)

func main() {
	fmt.Println("Avixer IO Usage Example")
	fmt.Println("=======================")

	// Example 1: Reading from a byte buffer
	fmt.Println("\n1. Reading AVI from memory buffer...")
	readFromBuffer()

	// Example 2: Writing to a byte buffer
	fmt.Println("\n2. Writing AVI to memory buffer...")
	writeToBuffer()

	// Example 3: Processing AVI streams
	fmt.Println("\n3. Processing AVI without files...")
	processInMemory()

	fmt.Println("\nExample completed!")
}

func readFromBuffer() {
	// Read file into memory
	data, err := os.ReadFile("../video.avi")
	if err != nil {
		fmt.Printf("Could not read video file: %v\n", err)
		fmt.Println("Make sure you have a video.avi file in the parent directory")
		return
	}

	// Create a bytes reader
	reader := bytes.NewReader(data)

	// Create demuxer
	demuxer := avi.NewDemuxer()
	defer demuxer.Close()

	// Open from reader
	err = demuxer.Open(reader, int64(len(data)))
	if err != nil {
		log.Printf("Failed to open from buffer: %v", err)
		return
	}

	// Get file information
	fileInfo, err := demuxer.GetFileInfo()
	if err != nil {
		log.Printf("Failed to get file info: %v", err)
		return
	}

	fmt.Printf("Loaded from buffer: %d bytes\n", fileInfo.FileSize)
	fmt.Printf("Duration: %v\n", fileInfo.Duration)
	fmt.Printf("Streams: %d video, %d audio\n", fileInfo.VideoStreams, fileInfo.AudioStreams)
}

func writeToBuffer() {
	// Create a seekable buffer for writing
	buffer := avi.NewSeekableBuffer()

	// Create muxer
	muxer := avi.NewMuxer()
	defer muxer.Close()

	// Create AVI in memory
	err := muxer.Create(buffer)
	if err != nil {
		log.Printf("Failed to create in buffer: %v", err)
		return
	}

	// Add video stream
	videoCodec := avi.Codec{
		Name:   "MJPG",
		FourCC: [4]byte{'M', 'J', 'P', 'G'},
		Type:   avi.StreamTypeVideo,
		Width:  320,
		Height: 240,
		FPS:    15.0,
	}

	videoIndex, err := muxer.AddStream(videoCodec)
	if err != nil {
		log.Printf("Failed to add video stream: %v", err)
		return
	}

	fmt.Printf("Added video stream #%d to buffer\n", videoIndex)

	// Write a few frames
	for i := 0; i < 5; i++ {
		// Create dummy frame data
		frameData := make([]byte, 500+i*50)
		
		packet := &avi.Packet{
			StreamIndex: videoIndex,
			Codec:       avi.StreamTypeVideo,
			Data:        frameData,
			PTS:         int64(i),
			DTS:         int64(i),
			Duration:    1,
			Flags:       "K__",
		}

		if i > 0 {
			packet.Flags = "___"
		}

		err = muxer.WritePacket(packet)
		if err != nil {
			log.Printf("Failed to write packet %d: %v", i, err)
			return
		}
	}

	// Finalize
	err = muxer.Finalize()
	if err != nil {
		log.Printf("Failed to finalize: %v", err)
		return
	}

	fmt.Printf("Created AVI in memory: %d bytes\n", buffer.Len())
}

func processInMemory() {
	// Read source file
	sourceData, err := os.ReadFile("../video.avi")
	if err != nil {
		fmt.Printf("Could not read source file: %v\n", err)
		return
	}

	// Open with demuxer
	demuxer := avi.NewDemuxer()
	defer demuxer.Close()

	reader := bytes.NewReader(sourceData)
	err = demuxer.Open(reader, int64(len(sourceData)))
	if err != nil {
		log.Printf("Failed to open source: %v", err)
		return
	}

	// Get streams
	streams, err := demuxer.GetStreams()
	if err != nil {
		log.Printf("Failed to get streams: %v", err)
		return
	}

	// Create output buffer
	outputBuffer := avi.NewSeekableBuffer()

	// Create muxer
	muxer := avi.NewMuxer()
	defer muxer.Close()

	err = muxer.Create(outputBuffer)
	if err != nil {
		log.Printf("Failed to create output: %v", err)
		return
	}

	// Copy streams
	for _, stream := range streams {
		_, err := muxer.AddStream(stream.Codec)
		if err != nil {
			log.Printf("Failed to add stream: %v", err)
			return
		}
	}

	// In a real application, you would read packets from demuxer
	// and write them to muxer after processing

	fmt.Printf("Set up processing pipeline with %d streams\n", len(streams))
	fmt.Println("Ready to process packets in memory (not implemented in this example)")
}