# Avixer - AVI Muxer and Demuxer Library

⚠️ **WARNING: This code has been generated with Claude AI and is NOT production ready. Use at your own risk for educational or experimental purposes only.**

Avixer is a Go library and command-line tool for reading and writing AVI (Audio Video Interleave) files. It provides easy-to-use interfaces for both library usage and command-line operations.

## Features

- **AVI Demuxer**: Read and parse AVI files, extract metadata and stream information
- **AVI Muxer**: Create new AVI files with video and audio streams
- **CLI Tool**: Command-line interface similar to ffprobe for analyzing AVI files
- **JSON Output**: Generate detailed JSON metadata files
- **Stream Support**: Handle both video and audio streams
- **Go Library**: Easy-to-use interfaces for Go projects

## Installation

```bash
go install github.com/avixer/avixer/cmd/avixer@latest
```

Or build from source:

```bash
git clone https://github.com/avixer/avixer
cd avixer
go build ./cmd/avixer
```

## Command Line Usage

### Basic Analysis

Analyze an AVI file and generate a JSON metadata file:

```bash
avixer -i video.avi
```

This creates `video.avi.json` with detailed file information.

### Options

```bash
avixer [options] -i input.avi

Options:
  -i string        Input AVI file (required)
  -o string        Output file (default: input.avi.json)
  -f string        Output format: json, text (default: json)
  -show-streams    Show stream information (default: true)
  -show-packets    Show packet information (default: false)
  -v               Verbose output
```

### Examples

```bash
# Analyze video.avi, output to video.avi.json
avixer -i video.avi

# Custom output file
avixer -i video.avi -o metadata.json

# Text output instead of JSON
avixer -i video.avi -f text

# Include packet information
avixer -i video.avi -show-packets

# Verbose output
avixer -i video.avi -v
```

## Library Usage

### Reading AVI Files (Demuxer)

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/avixer/avixer/avi"
)

func main() {
    // Create demuxer
    demuxer := avi.NewDemuxer()
    defer demuxer.Close()
    
    // Open file (convenience method)
    err := demuxer.OpenFile("video.avi")
    if err != nil {
        log.Fatal(err)
    }
    
    // Get file information
    fileInfo, err := demuxer.GetFileInfo()
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Duration: %v\n", fileInfo.Duration)
    fmt.Printf("Video streams: %d\n", fileInfo.VideoStreams)
    fmt.Printf("Audio streams: %d\n", fileInfo.AudioStreams)
    
    // Get stream details
    streams, err := demuxer.GetStreams()
    if err != nil {
        log.Fatal(err)
    }
    
    for _, stream := range streams {
        fmt.Printf("Stream %d: %s\n", stream.Index, stream.Type)
        if stream.Type == avi.StreamTypeVideo {
            fmt.Printf("  Resolution: %dx%d\n", stream.Codec.Width, stream.Codec.Height)
            fmt.Printf("  FPS: %.2f\n", stream.Codec.FPS)
        } else if stream.Type == avi.StreamTypeAudio {
            fmt.Printf("  Sample rate: %d Hz\n", stream.Codec.SampleRate)
            fmt.Printf("  Channels: %d\n", stream.Codec.Channels)
        }
    }
}
```

### Writing AVI Files (Muxer)

```go
package main

import (
    "log"
    
    "github.com/avixer/avixer/avi"
)

func main() {
    // Create muxer
    muxer := avi.NewMuxer()
    defer muxer.Close()
    
    // Create output file (convenience method)
    err := muxer.CreateFile("output.avi")
    if err != nil {
        log.Fatal(err)
    }
    
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
        log.Fatal(err)
    }
    
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
        log.Fatal(err)
    }
    
    // Write video frame
    videoPacket := &avi.Packet{
        StreamIndex: videoStreamIndex,
        Codec:       avi.StreamTypeVideo,
        Data:        []byte{/* your video frame data */},
        PTS:         0,
        DTS:         0,
        Duration:    1,
        Flags:       "K__", // keyframe
    }
    
    err = muxer.WritePacket(videoPacket)
    if err != nil {
        log.Fatal(err)
    }
    
    // Write audio sample
    audioPacket := &avi.Packet{
        StreamIndex: audioStreamIndex,
        Codec:       avi.StreamTypeAudio,
        Data:        []byte{/* your audio data */},
        PTS:         0,
        DTS:         0,
        Duration:    1024,
        Flags:       "K__",
    }
    
    err = muxer.WritePacket(audioPacket)
    if err != nil {
        log.Fatal(err)
    }
    
    // Finalize file (writes headers and indices)
    err = muxer.Finalize()
    if err != nil {
        log.Fatal(err)
    }
}
```

### Working with io.Reader/io.Writer

The library also supports working directly with `io.ReadSeeker` and `io.WriteSeeker` interfaces, allowing you to process AVI data in memory or from any source:

```go
package main

import (
    "bytes"
    "log"
    "os"
    
    "github.com/avixer/avixer/avi"
)

func main() {
    // Read from memory buffer
    data, err := os.ReadFile("video.avi")
    if err != nil {
        log.Fatal(err)
    }
    
    reader := bytes.NewReader(data)
    demuxer := avi.NewDemuxer()
    defer demuxer.Close()
    
    // Open from reader instead of file
    err = demuxer.Open(reader, int64(len(data)))
    if err != nil {
        log.Fatal(err)
    }
    
    // Write to memory buffer (using SeekableBuffer for io.WriteSeeker)
    buffer := avi.NewSeekableBuffer()
    muxer := avi.NewMuxer()
    defer muxer.Close()
    
    // Create in buffer instead of file
    err = muxer.Create(buffer)
    if err != nil {
        log.Fatal(err)
    }
    
    // ... add streams and write packets ...
    
    err = muxer.Finalize()
    if err != nil {
        log.Fatal(err)
    }
    
    // buffer now contains the complete AVI file
    aviData := buffer.Bytes()
}
```

## JSON Output Format

The CLI tool generates JSON files with the following structure:

```json
{
    "streams": [
        {
            "index": 0,
            "codec_type": "video",
            "codec_name": "MJPG",
            "width": 640,
            "height": 480,
            "fps": 30.0,
            "duration": "10.5s"
        },
        {
            "index": 1,
            "codec_type": "audio", 
            "codec_name": "PCM",
            "channels": 2,
            "sample_rate": 44100,
            "bit_depth": 16,
            "duration": "10.5s"
        }
    ],
    "packets": [
        {
            "codec_type": "video",
            "stream_index": 0,
            "dts": 0,
            "dts_time": "0.000000",
            "duration": 1,
            "duration_time": "0.033333",
            "size": "12253",
            "pos": "9992",
            "flags": "K__"
        }
    ]
}
```

## API Reference

### Types

- `StreamType`: Video or audio stream type
- `Codec`: Codec information (name, dimensions, sample rate, etc.)
- `Packet`: Media packet with data and timing information
- `Stream`: Stream metadata
- `FileInfo`: Overall file information

### Interfaces

- `Demuxer`: Interface for reading AVI files
- `Muxer`: Interface for writing AVI files

### Key Methods

**Demuxer:**
- `Open(filename string) error`
- `GetFileInfo() (*FileInfo, error)`
- `GetStreams() ([]Stream, error)`
- `ReadPacket() (*Packet, error)`
- `Close() error`

**Muxer:**
- `Create(filename string) error`
- `AddStream(codec Codec) (int, error)`
- `WritePacket(packet *Packet) error`
- `Finalize() error`
- `Close() error`

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License.