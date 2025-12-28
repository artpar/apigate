// Package streaming provides utilities for handling streaming protocols.
package streaming

import (
	"bytes"
	"io"
	"sync/atomic"
)

// StreamMetrics tracks metrics for a streaming response.
type StreamMetrics struct {
	TotalBytes   int64
	ChunkCount   int64
	LastChunk    []byte // Last chunk received (for metering from final data)
	AllData      []byte // Accumulated data (optional, can be disabled for large streams)
	AccumulateAll bool  // Whether to keep all data
}

// StreamReader wraps a reader to track streaming metrics.
type StreamReader struct {
	reader       io.ReadCloser
	totalBytes   atomic.Int64
	chunkCount   atomic.Int64
	lastChunk    []byte
	buffer       bytes.Buffer
	accumulate   bool
	closed       bool
}

// NewStreamReader creates a reader that tracks stream metrics.
// If accumulate is true, all data is kept for final metering (uses more memory).
// If accumulate is false, only the last chunk is kept.
func NewStreamReader(r io.ReadCloser, accumulate bool) *StreamReader {
	return &StreamReader{
		reader:     r,
		accumulate: accumulate,
	}
}

// Read implements io.Reader, tracking bytes as they pass through.
func (s *StreamReader) Read(p []byte) (int, error) {
	n, err := s.reader.Read(p)
	if n > 0 {
		s.totalBytes.Add(int64(n))
		s.chunkCount.Add(1)

		// Keep track of last chunk
		s.lastChunk = make([]byte, n)
		copy(s.lastChunk, p[:n])

		// Optionally accumulate all data
		if s.accumulate {
			s.buffer.Write(p[:n])
		}
	}
	return n, err
}

// Close closes the underlying reader.
func (s *StreamReader) Close() error {
	s.closed = true
	return s.reader.Close()
}

// GetMetrics returns the accumulated stream metrics.
func (s *StreamReader) GetMetrics() StreamMetrics {
	metrics := StreamMetrics{
		TotalBytes:    s.totalBytes.Load(),
		ChunkCount:    s.chunkCount.Load(),
		LastChunk:     s.lastChunk,
		AccumulateAll: s.accumulate,
	}
	if s.accumulate {
		metrics.AllData = s.buffer.Bytes()
	}
	return metrics
}

// GetTotalBytes returns the total bytes streamed.
func (s *StreamReader) GetTotalBytes() int64 {
	return s.totalBytes.Load()
}

// GetLastChunk returns the last chunk of data received.
func (s *StreamReader) GetLastChunk() []byte {
	return s.lastChunk
}

// GetAllData returns all accumulated data (only if accumulate was true).
func (s *StreamReader) GetAllData() []byte {
	if s.accumulate {
		return s.buffer.Bytes()
	}
	return nil
}

// TeeReader wraps a reader to copy data to a writer while reading.
// Useful for streaming responses while simultaneously accumulating for metering.
type TeeReader struct {
	reader io.ReadCloser
	writer io.Writer
	bytes  atomic.Int64
}

// NewTeeReader creates a reader that copies to writer while reading.
func NewTeeReader(r io.ReadCloser, w io.Writer) *TeeReader {
	return &TeeReader{
		reader: r,
		writer: w,
	}
}

// Read reads from the underlying reader and writes to the writer.
func (t *TeeReader) Read(p []byte) (int, error) {
	n, err := t.reader.Read(p)
	if n > 0 {
		t.bytes.Add(int64(n))
		if t.writer != nil {
			t.writer.Write(p[:n])
		}
	}
	return n, err
}

// Close closes the underlying reader.
func (t *TeeReader) Close() error {
	return t.reader.Close()
}

// GetBytes returns total bytes read.
func (t *TeeReader) GetBytes() int64 {
	return t.bytes.Load()
}

// ByteCounter wraps a reader to count bytes.
type ByteCounter struct {
	reader io.ReadCloser
	count  atomic.Int64
}

// NewByteCounter creates a byte counting reader wrapper.
func NewByteCounter(r io.ReadCloser) *ByteCounter {
	return &ByteCounter{reader: r}
}

// Read reads and counts bytes.
func (b *ByteCounter) Read(p []byte) (int, error) {
	n, err := b.reader.Read(p)
	if n > 0 {
		b.count.Add(int64(n))
	}
	return n, err
}

// Close closes the underlying reader.
func (b *ByteCounter) Close() error {
	return b.reader.Close()
}

// Count returns total bytes read.
func (b *ByteCounter) Count() int64 {
	return b.count.Load()
}
