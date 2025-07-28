# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Avixer is a Go library and CLI tool for reading and writing AVI (Audio Video Interleave) files. It was built to match ffprobe's JSON output format exactly and provides both library interfaces and a command-line tool.

## Development Commands

### Building
```bash
make build                 # Build CLI binary to build/avixer
make build-all            # Cross-compile for Linux, macOS, Windows (amd64/arm64)
go build ./cmd/avixer     # Direct build without Makefile
```

### Testing
```bash
make test                 # Run all tests
make test-coverage        # Generate coverage report (coverage.html)
go test ./avi -run TestDemuxerOpen  # Run specific test
make bench               # Run benchmarks
```

### Code Quality
```bash
make fmt                  # Format code with gofmt
make lint                 # Run golangci-lint (requires installation)
make security            # Run gosec security scan
```

### Development Workflow
```bash
make clean               # Remove build artifacts and test files
make deps               # Download dependencies
make tidy               # Tidy go.mod
make dev-setup          # Install dev tools (linter, security scanner)
```

### Testing with Real Files
```bash
# The CLI expects test files in data/ directory
./build/avixer -i data/video.avi                    # Analyze AVI file
./build/avixer -i data/video.avi -show-packets      # Include packet info
./build/avixer -i data/video.avi -f text           # Text output

# Run example
make example            # Builds and runs examples/basic_usage.go
```

## Architecture

### Core Library Structure

The library is organized into three main components:

1. **AVI Package (`/avi`)** - Core library functionality
   - `types.go` - Interfaces and data structures
   - `format.go` - AVI format constants and binary structures
   - `demuxer.go` - Reader implementation for parsing AVI files
   - `muxer.go` - Writer implementation for creating AVI files
   - `buffer.go` - SeekableBuffer for in-memory operations

2. **CLI Tool (`/cmd/avixer`)** - ffprobe-like command-line interface
   - Reads AVI files and outputs JSON/text metadata
   - Designed to match ffprobe's JSON output format exactly

3. **Examples (`/examples`)** - Usage demonstrations
   - `basic_usage.go` - File-based read/write operations
   - `io_usage.go` - Memory-based operations with io.Reader/Writer

### Key Design Decisions

1. **IO Flexibility**: The library supports both file operations and io.Reader/Writer interfaces:
   - `OpenFile(filename)` / `CreateFile(filename)` - Convenience methods
   - `Open(io.ReadSeeker, size)` / `Create(io.WriteSeeker)` - Generic IO interfaces

2. **Packet Reading**: The demuxer reads the AVI index (IDX1) to extract packet information with accurate:
   - Timestamps (PTS/DTS)
   - Byte positions
   - Sizes
   - Keyframe flags

3. **ffprobe Compatibility**: The JSON output format exactly matches ffprobe's structure:
   - Stream metadata (codec, dimensions, sample rate)
   - Packet information (timing, position, flags)
   - Duration formatting

### AVI File Processing Flow

**Demuxing (Reading)**:
1. Parse RIFF header → Verify AVI signature
2. Process LIST chunks → Extract stream headers (hdrl)
3. Store movi chunk offset → For packet position calculation
4. Parse IDX1 index → Build packet list with timestamps
5. Calculate timestamps → Based on stream properties (FPS, sample rate)

**Muxing (Writing)**:
1. Buffer packets in memory
2. On Finalize():
   - Write RIFF/AVI headers
   - Write stream headers (hdrl LIST)
   - Write packet data (movi LIST)
   - Write index (idx1 chunk)

### Important Implementation Details

- **Packet Timestamps**: 
  - Video: DTS = frame number, duration = 1 frame
  - Audio: DTS = sample count, duration = 1024 samples (typical for AVI)

- **Position Calculation**: `packet.Position = idx1.Offset + moviOffset + 4`
  - The +4 accounts for the 'movi' signature

- **Codec Name Cleaning**: Audio codec names are cleaned to remove non-printable characters

- **SeekableBuffer**: Custom implementation to provide io.WriteSeeker for in-memory operations

## Test Data

Test files should be placed in the `data/` directory. The tests look for:
- `data/video.avi` - Primary test file
- `../video.avi` - Alternative location for some tests

The project was developed using a specific test video.avi file (1920x1080, H.264 video, 8kHz audio, ~60 seconds).

## JSON Output Format

The CLI produces JSON matching ffprobe's structure:
```json
{
  "streams": [...],      // Stream metadata
  "packets": [...]       // Packet information (with -show-packets)
}
```

Key fields match ffprobe exactly: codec_type, stream_index, pts, dts, duration, size, pos, flags.