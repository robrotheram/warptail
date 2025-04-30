package logs

import (
	"bytes"
	"os"
	"sync"
)

// DualWriter writes to both a file and an in-memory buffer.
type DualWriter struct {
	file   *os.File
	buffer *bytes.Buffer
	mu     sync.Mutex
}

// NewDualWriter creates a new DualWriter for the given file path.
func NewDualWriter(filePath string) (*DualWriter, error) {
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &DualWriter{
		file:   f,
		buffer: &bytes.Buffer{},
	}, nil
}

// Write writes data to both the file and the buffer.
func (dw *DualWriter) Write(p []byte) (n int, err error) {
	dw.mu.Lock()
	defer dw.mu.Unlock()
	n, err = dw.file.Write(p)
	if err != nil {
		return n, err
	}
	_, _ = dw.buffer.Write(p)
	return n, nil
}

func (dw *DualWriter) WriteString(str string) (n int, err error) {
	return dw.Write([]byte(str))
}

func (dw *DualWriter) Flush() error {
	dw.mu.Lock()
	defer dw.mu.Unlock()
	if err := dw.file.Sync(); err != nil {
		return err
	}
	return nil
}

// ReadBuffer returns the current contents of the buffer as a string.
func (dw *DualWriter) ReadBuffer() string {
	dw.mu.Lock()
	defer dw.mu.Unlock()
	return dw.buffer.String()
}

func (dw *DualWriter) Read(p []byte) (n int, err error) {
	dw.mu.Lock()
	defer dw.mu.Unlock()
	return dw.buffer.Read(p)
}

// Close closes the underlying file.
func (dw *DualWriter) Close() error {
	return dw.file.Close()
}
