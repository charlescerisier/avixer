package avi

import (
	"bytes"
	"errors"
	"io"
)

// SeekableBuffer wraps bytes.Buffer to implement io.WriteSeeker
type SeekableBuffer struct {
	buf *bytes.Buffer
	pos int64
}

// NewSeekableBuffer creates a new SeekableBuffer
func NewSeekableBuffer() *SeekableBuffer {
	return &SeekableBuffer{
		buf: new(bytes.Buffer),
		pos: 0,
	}
}

// Write writes data to the buffer at the current position
func (sb *SeekableBuffer) Write(p []byte) (n int, err error) {
	// If we're not at the end, we need to handle overwriting
	currentLen := int64(sb.buf.Len())
	if sb.pos < currentLen {
		// Get current buffer data
		currentData := sb.buf.Bytes()
		
		// Create new buffer with data before position
		newBuf := new(bytes.Buffer)
		newBuf.Write(currentData[:sb.pos])
		
		// Write new data
		n, err = newBuf.Write(p)
		if err != nil {
			return n, err
		}
		
		// Write remaining data after the new data
		endPos := sb.pos + int64(n)
		if endPos < currentLen {
			newBuf.Write(currentData[endPos:])
		}
		
		sb.buf = newBuf
	} else {
		// Writing at or past the end
		n, err = sb.buf.Write(p)
	}
	
	sb.pos += int64(n)
	return n, err
}

// Seek sets the position for the next Write operation
func (sb *SeekableBuffer) Seek(offset int64, whence int) (int64, error) {
	var newPos int64
	
	switch whence {
	case io.SeekStart:
		newPos = offset
	case io.SeekCurrent:
		newPos = sb.pos + offset
	case io.SeekEnd:
		newPos = int64(sb.buf.Len()) + offset
	default:
		return 0, errors.New("invalid seek whence")
	}
	
	if newPos < 0 {
		return 0, errors.New("seek before start of buffer")
	}
	
	// If seeking past the end, we need to pad with zeros
	currentLen := int64(sb.buf.Len())
	if newPos > currentLen {
		padding := make([]byte, newPos-currentLen)
		sb.buf.Write(padding)
	}
	
	sb.pos = newPos
	return newPos, nil
}

// Read implements io.Reader for convenience
func (sb *SeekableBuffer) Read(p []byte) (n int, err error) {
	if sb.pos >= int64(sb.buf.Len()) {
		return 0, io.EOF
	}
	
	n = copy(p, sb.buf.Bytes()[sb.pos:])
	sb.pos += int64(n)
	
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

// Bytes returns the buffer contents
func (sb *SeekableBuffer) Bytes() []byte {
	return sb.buf.Bytes()
}

// Len returns the buffer length
func (sb *SeekableBuffer) Len() int {
	return sb.buf.Len()
}

// Reset resets the buffer
func (sb *SeekableBuffer) Reset() {
	sb.buf.Reset()
	sb.pos = 0
}