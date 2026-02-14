package logger

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"
)

const ringBufferSize = 1000

// Log is the global logger instance.
var Log zerolog.Logger

// LogRing holds the last 1000 log lines for the in-app log viewer. Set by Init.
var LogRing *RingBuffer

// RingBuffer is a thread-safe ring buffer that holds the last N log lines for in-app log viewer.
// It implements io.Writer.
type RingBuffer struct {
	mu     sync.RWMutex
	lines  []string
	index  int
	filled bool
}

// Write implements io.Writer. Each write is treated as one or more log lines (split by newline).
func (r *RingBuffer) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	s := strings.TrimRight(string(p), "\n\r")
	lines := strings.Split(s, "\n")
	r.mu.Lock()
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if len(r.lines) < ringBufferSize {
			r.lines = append(r.lines, line)
		} else {
			r.lines[r.index] = line
			r.index = (r.index + 1) % ringBufferSize
			r.filled = true
		}
	}
	r.mu.Unlock()
	return len(p), nil
}

// Lines returns all buffered log lines in chronological order.
func (r *RingBuffer) Lines() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if !r.filled {
		result := make([]string, len(r.lines))
		copy(result, r.lines)
		return result
	}
	result := make([]string, ringBufferSize)
	for i := 0; i < ringBufferSize; i++ {
		result[i] = r.lines[(r.index+i)%ringBufferSize]
	}
	return result
}

// Init initializes the global logger. It writes to:
// - A rotating file at logDir/cs2admin.log
// - An in-memory ring buffer (last 1000 lines)
// - Console with pretty colored output when debug is true
//
// Creates logDir if it does not exist.
func Init(logDir string, debug bool) error {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	ring := &RingBuffer{
		lines: make([]string, 0, ringBufferSize),
	}
	LogRing = ring

	fileWriter := &lumberjack.Logger{
		Filename:   filepath.Join(logDir, "cs2admin.log"),
		MaxSize:    10,
		MaxBackups: 3,
		MaxAge:     28,
		Compress:   true,
	}

	writers := []io.Writer{fileWriter, ring}
	if debug {
		console := zerolog.ConsoleWriter{Out: os.Stderr, NoColor: false}
		writers = append(writers, console)
	}

	multi := zerolog.MultiLevelWriter(writers...)
	Log = zerolog.New(multi).With().Timestamp().Logger()
	return nil
}
