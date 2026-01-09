package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/charlescerisier/avixer/avi"
)

// Config holds CLI configuration
type Config struct {
	InputFile  string
	OutputFile string
	Verbose    bool
	Progress   bool
	DryRun     bool
}

// Version can be set at build time
var version = "dev"

func main() {
	config := parseFlags()

	if config.InputFile == "" {
		fmt.Fprintf(os.Stderr, "Error: input file is required\n")
		flag.Usage()
		os.Exit(1)
	}

	// Check if input file exists
	if _, err := os.Stat(config.InputFile); os.IsNotExist(err) {
		log.Fatalf("Error: input file '%s' does not exist", config.InputFile)
	}

	// Set default output file if not specified
	if config.OutputFile == "" {
		dir := filepath.Dir(config.InputFile)
		base := filepath.Base(config.InputFile)
		ext := filepath.Ext(base)
		name := base[:len(base)-len(ext)]
		config.OutputFile = filepath.Join(dir, name+"_remuxed"+ext)
	}

	// Perform remuxing
	if err := remuxFile(config); err != nil {
		log.Fatalf("Error remuxing file: %v", err)
	}
}

func parseFlags() Config {
	var config Config

	flag.StringVar(&config.InputFile, "i", "", "Input AVI file (required)")
	flag.StringVar(&config.OutputFile, "o", "", "Output AVI file (default: input_remuxed.avi)")
	flag.BoolVar(&config.Verbose, "v", false, "Verbose output")
	flag.BoolVar(&config.Progress, "p", false, "Show progress")
	flag.BoolVar(&config.DryRun, "dry-run", false, "Analyze input without creating output")

	var showVersion bool
	flag.BoolVar(&showVersion, "version", false, "Show version")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "aviremux %s - AVI file remuxer\n", version)
		fmt.Fprintf(os.Stderr, "\nUsage: %s [options] -i input.avi\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s -i video.avi                    # Remux to video_remuxed.avi\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -i video.avi -o output.avi      # Specify output file\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -i video.avi -v -p              # Verbose with progress\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -i video.avi --dry-run          # Analyze without remuxing\n", os.Args[0])
	}

	flag.Parse()

	if showVersion {
		fmt.Printf("aviremux %s\n", version)
		os.Exit(0)
	}

	return config
}

func remuxFile(config Config) error {
	startTime := time.Now()

	// Open input file
	if config.Verbose {
		fmt.Printf("Opening input file: %s\n", config.InputFile)
	}

	demuxer := avi.NewDemuxer()
	defer demuxer.Close()

	if err := demuxer.OpenFile(config.InputFile); err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}

	// Get file information
	fileInfo, err := demuxer.GetFileInfo()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	// Get streams
	streams, err := demuxer.GetStreams()
	if err != nil {
		return fmt.Errorf("failed to get streams: %w", err)
	}

	if config.Verbose || config.DryRun {
		fmt.Printf("\nInput file information:\n")
		fmt.Printf("  File: %s\n", filepath.Base(config.InputFile))
		fmt.Printf("  Size: %s\n", formatBytes(fileInfo.FileSize))
		fmt.Printf("  Duration: %v\n", fileInfo.Duration)
		fmt.Printf("  Streams: %d video, %d audio\n", fileInfo.VideoStreams, fileInfo.AudioStreams)

		fmt.Printf("\nStream details:\n")
		for _, stream := range streams {
			fmt.Printf("  Stream #%d: %s\n", stream.Index, stream.Type)
			if stream.Type == avi.StreamTypeVideo {
				fmt.Printf("    Codec: %s\n", stream.Codec.Name)
				fmt.Printf("    Resolution: %dx%d @ %.2f fps\n",
					stream.Codec.Width, stream.Codec.Height, stream.Codec.FPS)
			} else if stream.Type == avi.StreamTypeAudio {
				fmt.Printf("    Codec: %s\n", formatCodecName(stream.Codec.Name))
				fmt.Printf("    Format: %d Hz, %d channels, %d bit\n",
					stream.Codec.SampleRate, stream.Codec.Channels, stream.Codec.BitDepth)
			}
			fmt.Printf("    Duration: %v\n", stream.Duration)
		}
	}

	// Read all packets
	reader, ok := demuxer.(*avi.Reader)
	if !ok {
		return fmt.Errorf("internal error: demuxer is not a Reader")
	}

	if config.Verbose {
		fmt.Printf("\nReading packets...\n")
	}

	packets, err := reader.ReadAllPackets()
	if err != nil {
		return fmt.Errorf("failed to read packets: %w", err)
	}

	if config.Verbose || config.DryRun {
		fmt.Printf("  Total packets: %d\n", len(packets))

		// Count packets per stream
		streamPacketCounts := make(map[int]int)
		totalSize := int64(0)
		for _, packet := range packets {
			streamPacketCounts[packet.StreamIndex]++
			totalSize += int64(packet.Size)
		}

		for i := 0; i < len(streams); i++ {
			if count, ok := streamPacketCounts[i]; ok {
				fmt.Printf("  Stream #%d: %d packets\n", i, count)
			}
		}
		fmt.Printf("  Total data size: %s\n", formatBytes(totalSize))
	}

	if config.DryRun {
		fmt.Printf("\nDry run complete. No output file created.\n")
		return nil
	}

	// Create output file
	if config.Verbose {
		fmt.Printf("\nCreating output file: %s\n", config.OutputFile)
	}

	muxer := avi.NewMuxer()
	defer muxer.Close()

	if err := muxer.CreateFile(config.OutputFile); err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}

	// Add streams to output
	streamMapping := make(map[int]int)
	for _, stream := range streams {
		newIndex, err := muxer.AddStream(stream.Codec)
		if err != nil {
			return fmt.Errorf("failed to add stream: %w", err)
		}
		streamMapping[stream.Index] = newIndex
		if config.Verbose {
			fmt.Printf("  Added stream #%d (%s) -> #%d\n", stream.Index, stream.Type, newIndex)
		}
	}

	// Write packets
	if config.Verbose || config.Progress {
		fmt.Printf("\nWriting packets...\n")
	}

	// Read and write the actual packet data
	for i, packet := range packets {
		// Read the actual packet data from the source file
		packetData, err := reader.ReadPacketData(&packet)
		if err != nil {
			return fmt.Errorf("failed to read packet %d data: %w", i, err)
		}
		
		// Create new packet with remapped stream index and real data
		newPacket := &avi.Packet{
			StreamIndex:  streamMapping[packet.StreamIndex],
			Codec:        packet.Codec,
			Data:         packetData,
			PTS:          packet.PTS,
			DTS:          packet.DTS,
			Duration:     packet.Duration,
			Size:         packet.Size,
			Flags:        packet.Flags,
			PTSTime:      packet.PTSTime,
			DTSTime:      packet.DTSTime,
			DurationTime: packet.DurationTime,
		}

		if err := muxer.WritePacket(newPacket); err != nil {
			return fmt.Errorf("failed to write packet %d: %w", i, err)
		}

		// Show progress
		if config.Progress && (i+1)%100 == 0 {
			progress := float64(i+1) / float64(len(packets)) * 100
			fmt.Printf("\r  Progress: %d/%d packets (%.1f%%)", i+1, len(packets), progress)
		}
	}

	if config.Progress {
		fmt.Printf("\r  Progress: %d/%d packets (100.0%%)\n", len(packets), len(packets))
	}

	// Finalize output
	if config.Verbose {
		fmt.Printf("\nFinalizing output file...\n")
	}

	if err := muxer.Finalize(); err != nil {
		return fmt.Errorf("failed to finalize output: %w", err)
	}

	// Get output file size
	outputInfo, err := os.Stat(config.OutputFile)
	if err != nil {
		return fmt.Errorf("failed to stat output file: %w", err)
	}

	elapsed := time.Since(startTime)

	// Summary
	fmt.Printf("\n✅ Remuxing completed successfully!\n")
	fmt.Printf("\nSummary:\n")
	fmt.Printf("  Input:  %s (%s)\n", filepath.Base(config.InputFile), formatBytes(fileInfo.FileSize))
	fmt.Printf("  Output: %s (%s)\n", filepath.Base(config.OutputFile), formatBytes(outputInfo.Size()))
	fmt.Printf("  Streams: %d\n", len(streams))
	fmt.Printf("  Packets: %d\n", len(packets))
	fmt.Printf("  Time: %v\n", elapsed)

	return nil
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func formatCodecName(name string) string {
	if name == "" {
		return "(none)"
	}
	return name
}