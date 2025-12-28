package streaming_test

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/artpar/apigate/domain/streaming"
)

type nopCloser struct {
	io.Reader
}

func (nopCloser) Close() error { return nil }

func TestStreamReader_BasicReading(t *testing.T) {
	data := "Hello, World!"
	reader := streaming.NewStreamReader(nopCloser{strings.NewReader(data)}, false)

	buf := make([]byte, 1024)
	n, err := reader.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("Read error: %v", err)
	}

	if n != len(data) {
		t.Errorf("read %d bytes, want %d", n, len(data))
	}

	if string(buf[:n]) != data {
		t.Errorf("got %q, want %q", string(buf[:n]), data)
	}
}

func TestStreamReader_TotalBytes(t *testing.T) {
	data := "0123456789" // 10 bytes
	reader := streaming.NewStreamReader(nopCloser{strings.NewReader(data)}, false)

	buf := make([]byte, 5)

	// Read first 5 bytes
	n1, _ := reader.Read(buf)
	if n1 != 5 {
		t.Errorf("first read: %d bytes, want 5", n1)
	}

	// Read next 5 bytes
	n2, _ := reader.Read(buf)
	if n2 != 5 {
		t.Errorf("second read: %d bytes, want 5", n2)
	}

	// Check total
	total := reader.GetTotalBytes()
	if total != 10 {
		t.Errorf("total bytes = %d, want 10", total)
	}
}

func TestStreamReader_ChunkCount(t *testing.T) {
	data := strings.Repeat("x", 100)
	reader := streaming.NewStreamReader(nopCloser{strings.NewReader(data)}, false)

	buf := make([]byte, 10)

	// Read in 10-byte chunks
	for {
		_, err := reader.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Read error: %v", err)
		}
	}

	metrics := reader.GetMetrics()
	if metrics.ChunkCount != 10 {
		t.Errorf("chunk count = %d, want 10", metrics.ChunkCount)
	}
}

func TestStreamReader_LastChunk(t *testing.T) {
	data := "first chunk|second chunk|last chunk"
	reader := streaming.NewStreamReader(nopCloser{strings.NewReader(data)}, false)

	buf := make([]byte, 12)

	// Read all chunks
	for {
		_, err := reader.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Read error: %v", err)
		}
	}

	lastChunk := reader.GetLastChunk()
	// Last chunk contains the remaining bytes
	if len(lastChunk) == 0 {
		t.Error("expected non-empty last chunk")
	}
}

func TestStreamReader_Accumulate(t *testing.T) {
	data := "part1|part2|part3"
	reader := streaming.NewStreamReader(nopCloser{strings.NewReader(data)}, true) // accumulate = true

	buf := make([]byte, 5)

	// Read all data
	for {
		_, err := reader.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Read error: %v", err)
		}
	}

	allData := reader.GetAllData()
	if string(allData) != data {
		t.Errorf("accumulated data = %q, want %q", string(allData), data)
	}

	metrics := reader.GetMetrics()
	if !metrics.AccumulateAll {
		t.Error("AccumulateAll should be true")
	}
	if string(metrics.AllData) != data {
		t.Error("metrics.AllData should contain all data")
	}
}

func TestStreamReader_NoAccumulate(t *testing.T) {
	data := "data that should not accumulate"
	reader := streaming.NewStreamReader(nopCloser{strings.NewReader(data)}, false) // accumulate = false

	buf := make([]byte, 1024)
	reader.Read(buf)

	allData := reader.GetAllData()
	if allData != nil {
		t.Error("GetAllData should return nil when not accumulating")
	}

	metrics := reader.GetMetrics()
	if metrics.AccumulateAll {
		t.Error("AccumulateAll should be false")
	}
}

func TestStreamReader_Close(t *testing.T) {
	data := "test data"
	reader := streaming.NewStreamReader(nopCloser{strings.NewReader(data)}, false)

	err := reader.Close()
	if err != nil {
		t.Errorf("Close error: %v", err)
	}
}

func TestStreamReader_Metrics(t *testing.T) {
	data := strings.Repeat("x", 50)
	reader := streaming.NewStreamReader(nopCloser{strings.NewReader(data)}, true)

	buf := make([]byte, 10)
	for {
		_, err := reader.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Read error: %v", err)
		}
	}

	metrics := reader.GetMetrics()

	if metrics.TotalBytes != 50 {
		t.Errorf("TotalBytes = %d, want 50", metrics.TotalBytes)
	}
	if metrics.ChunkCount != 5 {
		t.Errorf("ChunkCount = %d, want 5", metrics.ChunkCount)
	}
	if len(metrics.LastChunk) != 10 {
		t.Errorf("LastChunk length = %d, want 10", len(metrics.LastChunk))
	}
	if !metrics.AccumulateAll {
		t.Error("AccumulateAll should be true")
	}
	if len(metrics.AllData) != 50 {
		t.Errorf("AllData length = %d, want 50", len(metrics.AllData))
	}
}

func TestTeeReader_BasicReading(t *testing.T) {
	data := "Hello, TeeReader!"
	var buf bytes.Buffer
	reader := streaming.NewTeeReader(nopCloser{strings.NewReader(data)}, &buf)

	readBuf := make([]byte, 1024)
	n, err := reader.Read(readBuf)
	if err != nil && err != io.EOF {
		t.Fatalf("Read error: %v", err)
	}

	// Check read buffer
	if string(readBuf[:n]) != data {
		t.Errorf("read = %q, want %q", string(readBuf[:n]), data)
	}

	// Check tee'd data
	if buf.String() != data {
		t.Errorf("tee'd = %q, want %q", buf.String(), data)
	}
}

func TestTeeReader_ByteCount(t *testing.T) {
	data := strings.Repeat("x", 100)
	var buf bytes.Buffer
	reader := streaming.NewTeeReader(nopCloser{strings.NewReader(data)}, &buf)

	readBuf := make([]byte, 10)
	for {
		_, err := reader.Read(readBuf)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Read error: %v", err)
		}
	}

	if reader.GetBytes() != 100 {
		t.Errorf("GetBytes = %d, want 100", reader.GetBytes())
	}
}

func TestTeeReader_NilWriter(t *testing.T) {
	data := "test data"
	reader := streaming.NewTeeReader(nopCloser{strings.NewReader(data)}, nil)

	buf := make([]byte, 1024)
	n, err := reader.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("Read error: %v", err)
	}

	// Should still work with nil writer
	if string(buf[:n]) != data {
		t.Errorf("read = %q, want %q", string(buf[:n]), data)
	}
}

func TestTeeReader_Close(t *testing.T) {
	reader := streaming.NewTeeReader(nopCloser{strings.NewReader("test")}, nil)

	err := reader.Close()
	if err != nil {
		t.Errorf("Close error: %v", err)
	}
}

func TestByteCounter_BasicReading(t *testing.T) {
	data := "Count these bytes!"
	counter := streaming.NewByteCounter(nopCloser{strings.NewReader(data)})

	buf := make([]byte, 1024)
	n, err := counter.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("Read error: %v", err)
	}

	if string(buf[:n]) != data {
		t.Errorf("read = %q, want %q", string(buf[:n]), data)
	}

	if counter.Count() != int64(len(data)) {
		t.Errorf("Count = %d, want %d", counter.Count(), len(data))
	}
}

func TestByteCounter_MultipleReads(t *testing.T) {
	data := strings.Repeat("x", 100)
	counter := streaming.NewByteCounter(nopCloser{strings.NewReader(data)})

	buf := make([]byte, 10)
	totalRead := 0
	for {
		n, err := counter.Read(buf)
		totalRead += n
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Read error: %v", err)
		}
	}

	if counter.Count() != 100 {
		t.Errorf("Count = %d, want 100", counter.Count())
	}
	if totalRead != 100 {
		t.Errorf("totalRead = %d, want 100", totalRead)
	}
}

func TestByteCounter_Close(t *testing.T) {
	counter := streaming.NewByteCounter(nopCloser{strings.NewReader("test")})

	err := counter.Close()
	if err != nil {
		t.Errorf("Close error: %v", err)
	}
}

func TestByteCounter_EmptyRead(t *testing.T) {
	counter := streaming.NewByteCounter(nopCloser{strings.NewReader("")})

	buf := make([]byte, 10)
	n, err := counter.Read(buf)

	if n != 0 {
		t.Errorf("n = %d, want 0", n)
	}
	if err != io.EOF {
		t.Errorf("err = %v, want EOF", err)
	}
	if counter.Count() != 0 {
		t.Errorf("Count = %d, want 0", counter.Count())
	}
}

// Concurrent access tests
func TestStreamReader_ConcurrentReads(t *testing.T) {
	data := strings.Repeat("x", 1000)
	reader := streaming.NewStreamReader(nopCloser{strings.NewReader(data)}, true)

	done := make(chan bool)

	// Multiple goroutines reading metrics
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_ = reader.GetTotalBytes()
				_ = reader.GetMetrics()
			}
			done <- true
		}()
	}

	// One goroutine reading
	go func() {
		buf := make([]byte, 10)
		for {
			_, err := reader.Read(buf)
			if err != nil {
				break
			}
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 11; i++ {
		<-done
	}

	// Should complete without race conditions
	if reader.GetTotalBytes() != 1000 {
		t.Errorf("TotalBytes = %d, want 1000", reader.GetTotalBytes())
	}
}
