package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/avixer/avixer/avi"
)

// OutputFormat represents different output formats
type OutputFormat string

const (
	OutputJSON OutputFormat = "json"
	OutputText OutputFormat = "text"
)

// Config holds CLI configuration
type Config struct {
	InputFile    string
	OutputFile   string
	OutputFormat OutputFormat
	ShowStreams  bool
	ShowPackets  bool
	Verbose      bool
}

// PacketInfo represents packet information for JSON output
type PacketInfo struct {
	CodecType    string `json:"codec_type"`
	StreamIndex  int    `json:"stream_index"`
	PTS          *int64 `json:"pts,omitempty"`
	PTSTime      string `json:"pts_time,omitempty"`
	DTS          int64  `json:"dts"`
	DTSTime      string `json:"dts_time"`
	Duration     int64  `json:"duration"`
	DurationTime string `json:"duration_time"`
	Size         string `json:"size"`
	Pos          string `json:"pos"`
	Flags        string `json:"flags"`
}

// StreamInfo represents stream information for JSON output  
type StreamInfo struct {
	Index     int                    `json:"index"`
	CodecType string                 `json:"codec_type"`
	CodecName string                 `json:"codec_name,omitempty"`
	Width     int                    `json:"width,omitempty"`
	Height    int                    `json:"height,omitempty"`
	FPS       float64                `json:"fps,omitempty"`
	Channels  int                    `json:"channels,omitempty"`
	SampleRate int                   `json:"sample_rate,omitempty"`
	BitDepth  int                    `json:"bit_depth,omitempty"`
	Duration  string                 `json:"duration,omitempty"`
	Tags      map[string]interface{} `json:"tags,omitempty"`
}

// FileOutput represents the complete file information for JSON output
type FileOutput struct {
	Streams []StreamInfo `json:"streams,omitempty"`
	Packets []PacketInfo `json:"packets,omitempty"`
}

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

	// Analyze the AVI file
	if err := analyzeFile(config); err != nil {
		log.Fatalf("Error analyzing file: %v", err)
	}
}

func parseFlags() Config {
	var config Config

	flag.StringVar(&config.InputFile, "i", "", "Input AVI file")
	flag.StringVar(&config.OutputFile, "o", "", "Output file (default: input.avi.json)")
	flag.BoolVar(&config.ShowStreams, "show-streams", true, "Show stream information")
	flag.BoolVar(&config.ShowPackets, "show-packets", false, "Show packet information")
	flag.BoolVar(&config.Verbose, "v", false, "Verbose output")

	var format string
	flag.StringVar(&format, "f", "json", "Output format (json, text)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] -i input.avi\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s -i video.avi                    # Analyze video.avi, output to video.avi.json\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -i video.avi -o info.json       # Analyze video.avi, output to info.json\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -i video.avi -f text            # Text output instead of JSON\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -i video.avi -show-packets      # Include packet information\n", os.Args[0])
	}

	flag.Parse()

	// Set output format
	switch strings.ToLower(format) {
	case "json":
		config.OutputFormat = OutputJSON
	case "text":
		config.OutputFormat = OutputText
	default:
		log.Fatalf("Error: unsupported output format '%s'", format)
	}

	// Set default output file if not specified
	if config.OutputFile == "" && config.OutputFormat == OutputJSON {
		config.OutputFile = config.InputFile + ".json"
	}

	return config
}

func analyzeFile(config Config) error {
	// Create demuxer
	demuxer := avi.NewDemuxer()
	defer demuxer.Close()

	// Open file using convenience method
	if err := demuxer.OpenFile(config.InputFile); err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}

	// Get file info
	fileInfo, err := demuxer.GetFileInfo()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	// Get streams
	streams, err := demuxer.GetStreams()
	if err != nil {
		return fmt.Errorf("failed to get streams: %w", err)
	}

	if config.Verbose {
		fmt.Printf("Analyzing file: %s\n", config.InputFile)
		fmt.Printf("File size: %d bytes\n", fileInfo.FileSize)
		fmt.Printf("Duration: %v\n", fileInfo.Duration)
		fmt.Printf("Streams: %d video, %d audio\n", fileInfo.VideoStreams, fileInfo.AudioStreams)
	}

	// Generate output
	switch config.OutputFormat {
	case OutputJSON:
		return generateJSONOutput(config, fileInfo, streams, demuxer)
	case OutputText:
		return generateTextOutput(config, fileInfo, streams)
	default:
		return fmt.Errorf("unsupported output format")
	}
}

func generateJSONOutput(config Config, fileInfo *avi.FileInfo, streams []avi.Stream, demuxer avi.Demuxer) error {
	var output FileOutput

	// Add stream information
	if config.ShowStreams {
		for _, stream := range streams {
			streamInfo := StreamInfo{
				Index:     stream.Index,
				CodecType: string(stream.Type),
				CodecName: stream.Codec.Name,
				Duration:  stream.Duration.String(),
				Tags:      make(map[string]interface{}),
			}

			if stream.Type == avi.StreamTypeVideo {
				streamInfo.Width = stream.Codec.Width
				streamInfo.Height = stream.Codec.Height
				streamInfo.FPS = stream.Codec.FPS
			} else if stream.Type == avi.StreamTypeAudio {
				streamInfo.Channels = stream.Codec.Channels
				streamInfo.SampleRate = stream.Codec.SampleRate
				streamInfo.BitDepth = stream.Codec.BitDepth
			}

			output.Streams = append(output.Streams, streamInfo)
		}
	}

	// Add packet information from real file data
	if config.ShowPackets {
		packets, err := readRealPackets(demuxer)
		if err != nil {
			return fmt.Errorf("failed to read packets: %w", err)
		}
		output.Packets = convertPacketsToJSON(packets)
	}

	// Write output
	var err error
	if config.OutputFile != "" {
		err = writeJSONToFile(output, config.OutputFile)
	} else {
		err = writeJSONToStdout(output)
	}

	if err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

	if config.Verbose && config.OutputFile != "" {
		fmt.Printf("Output written to: %s\n", config.OutputFile)
	}

	return nil
}

func generateTextOutput(config Config, fileInfo *avi.FileInfo, streams []avi.Stream) error {
	// Write to stdout or file
	var output *os.File = os.Stdout

	if config.OutputFile != "" {
		var err error
		output, err = os.Create(config.OutputFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer output.Close()
	}

	// Write file information
	fmt.Fprintf(output, "File: %s\n", filepath.Base(fileInfo.Filename))
	fmt.Fprintf(output, "Size: %d bytes\n", fileInfo.FileSize)
	fmt.Fprintf(output, "Duration: %v\n", fileInfo.Duration)
	fmt.Fprintf(output, "Streams: %d video, %d audio\n\n", fileInfo.VideoStreams, fileInfo.AudioStreams)

	// Write stream information
	if config.ShowStreams {
		fmt.Fprintf(output, "Streams:\n")
		for _, stream := range streams {
			fmt.Fprintf(output, "  Stream #%d: %s", stream.Index, string(stream.Type))
			
			if stream.Type == avi.StreamTypeVideo {
				fmt.Fprintf(output, " (%s) %dx%d", stream.Codec.Name, stream.Codec.Width, stream.Codec.Height)
				if stream.Codec.FPS > 0 {
					fmt.Fprintf(output, " @ %.2f fps", stream.Codec.FPS)
				}
			} else if stream.Type == avi.StreamTypeAudio {
				fmt.Fprintf(output, " (%s) %d Hz, %d channels", stream.Codec.Name, stream.Codec.SampleRate, stream.Codec.Channels)
				if stream.Codec.BitDepth > 0 {
					fmt.Fprintf(output, ", %d bit", stream.Codec.BitDepth)
				}
			}
			
			if stream.Duration > 0 {
				fmt.Fprintf(output, ", duration: %v", stream.Duration)
			}
			fmt.Fprintf(output, "\n")
		}
	}

	return nil
}

func writeJSONToFile(output FileOutput, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "    ")
	return encoder.Encode(output)
}

func writeJSONToStdout(output FileOutput) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "    ")
	return encoder.Encode(output)
}

// readRealPackets reads actual packets from the AVI file
func readRealPackets(demuxer avi.Demuxer) ([]avi.Packet, error) {
	// Cast to *Reader to access ReadAllPackets method
	reader, ok := demuxer.(*avi.Reader)
	if !ok {
		return nil, fmt.Errorf("demuxer is not a Reader type")
	}
	
	return reader.ReadAllPackets()
}

// convertPacketsToJSON converts avi.Packet slice to PacketInfo slice for JSON output
func convertPacketsToJSON(packets []avi.Packet) []PacketInfo {
	var jsonPackets []PacketInfo
	
	for _, packet := range packets {
		jsonPacket := PacketInfo{
			CodecType:    string(packet.Codec),
			StreamIndex:  packet.StreamIndex,
			DTS:          packet.DTS,
			DTSTime:      fmt.Sprintf("%.6f", packet.DTSTime.Seconds()),
			Duration:     packet.Duration,
			DurationTime: fmt.Sprintf("%.6f", packet.DurationTime.Seconds()),
			Size:         fmt.Sprintf("%d", packet.Size),
			Pos:          fmt.Sprintf("%d", packet.Position),
			Flags:        packet.Flags,
		}
		
		// Add PTS for audio packets or when PTS != DTS  
		if packet.Codec == avi.StreamTypeAudio || packet.PTS != packet.DTS {
			pts := packet.PTS
			jsonPacket.PTS = &pts
			jsonPacket.PTSTime = fmt.Sprintf("%.6f", packet.PTSTime.Seconds())
		}
		
		jsonPackets = append(jsonPackets, jsonPacket)
	}
	
	return jsonPackets
}