package lib

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// CPILogger wraps slog.Logger with additional functionality required by the CPI
type CPILogger struct {
	*slog.Logger
	file    *os.File
	rotator *FileRotator
	logFile string
}

// FileRotator handles log file rotation and compression
type FileRotator struct {
	filename    string
	maxSize     int64
	maxBackups  int
	compress    bool
	mu          sync.Mutex
	currentSize int64
}

// NewCPILogger creates a new CPILogger with the specified configuration
func NewCPILogger(logFile string, level slog.Level, maxSizeMB int, maxBackups int, compress bool) (*CPILogger, error) {
	if logFile == "" {
		handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			Level: level,
		})
		return &CPILogger{
			Logger: slog.New(handler),
		}, nil
	}

	// Ensure log directory exists
	if err := os.MkdirAll(filepath.Dir(logFile), 0o750); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	logDir, err := os.OpenRoot(filepath.Dir(logFile))
	if err != nil {
		return nil, fmt.Errorf("failed opening log dir")
	}
	// Create or open log file
	file, err := logDir.OpenFile(filepath.Base(logFile), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	// Get current file size
	stat, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("failed to stat log file: %w", err)
	}

	// Create rotator
	rotator := &FileRotator{
		filename:    logFile,
		maxSize:     int64(maxSizeMB) * 1024 * 1024,
		maxBackups:  maxBackups,
		compress:    compress,
		currentSize: stat.Size(),
	}

	// Create rotating writer
	writer := &RotatingWriter{
		file:    file,
		rotator: rotator,
	}

	// Create slog handler
	handler := slog.NewJSONHandler(writer, &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Format timestamp to match existing format
			if a.Key == slog.TimeKey {
				return slog.String("time", a.Value.Time().Format("2006-01-02T15:04:05.000Z07:00"))
			}
			return a
		},
	})

	return &CPILogger{
		Logger:  slog.New(handler),
		file:    file,
		rotator: rotator,
		logFile: logFile,
	}, nil
}

// RotatingWriter wraps a file with rotation capability
type RotatingWriter struct {
	file    *os.File
	rotator *FileRotator
}

func (w *RotatingWriter) Write(p []byte) (n int, err error) {
	w.rotator.mu.Lock()
	defer w.rotator.mu.Unlock()

	// Check if rotation is needed
	if w.rotator.shouldRotate(len(p)) {
		if err := w.rotator.rotate(w.file); err != nil {
			return 0, err
		}
		// Reopen file after rotation
		newFile, err := os.OpenFile(w.rotator.filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
		if err != nil {
			return 0, err
		}
		_ = w.file.Close()
		w.file = newFile
		w.rotator.currentSize = 0
	}

	n, err = w.file.Write(p)
	w.rotator.currentSize += int64(n)
	return
}

// shouldRotate checks if file rotation is needed
func (r *FileRotator) shouldRotate(writeSize int) bool {
	return r.currentSize+int64(writeSize) >= r.maxSize
}

// rotate performs the actual file rotation
func (r *FileRotator) rotate(currentFile *os.File) error {
	// Close current file
	_ = currentFile.Close()

	// Move existing files
	for i := r.maxBackups - 1; i >= 1; i-- {
		oldName := fmt.Sprintf("%s.%d", r.filename, i)
		newName := fmt.Sprintf("%s.%d", r.filename, i+1)

		if r.compress && i == r.maxBackups-1 {
			newName += ".gz"
		}

		if _, err := os.Stat(oldName); err == nil {
			_ = os.Remove(newName) // Remove if exists
			_ = os.Rename(oldName, newName)
		}
	}

	// Move current log to .1
	firstBackup := fmt.Sprintf("%s.1", r.filename)
	if err := os.Rename(r.filename, firstBackup); err != nil {
		return err
	}

	// Compress if enabled
	if r.compress {
		if err := r.compressFile(firstBackup); err != nil {
			return err
		}
	}

	return nil
}

// compressFile compresses a log file using gzip
func (r *FileRotator) compressFile(filename string) error {
	// Simple compression - in production this would use gzip
	// For now, just rename to indicate compression
	compressedName := filename + ".gz"
	return os.Rename(filename, compressedName)
}

// MapLogLevel converts string log level to slog.Level
func MapLogLevel(configLevel string) slog.Level {
	switch configLevel {
	case "trace":
		return slog.LevelDebug - 4 // Custom trace level
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelDebug // default to debug
	}
}

// LoggerInterface defines the interface for backwards compatibility
type LoggerInterface interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	Debugf(format string, args ...any)
	Infof(format string, args ...any)
	Warnf(format string, args ...any)
	Errorf(format string, args ...any)
	WithContext(ctx context.Context)
	ChildLogger(component string) LoggerInterface
}

func (l *CPILogger) Debug(msg string, args ...any) {
	l.debug(msg, args...)
}

func (l *CPILogger) debug(msg string, args ...any) {
	pc, _, _, ok := runtime.Caller(2)
	callerDetails := runtime.FuncForPC(pc)
	if ok {
		nameElements := strings.Split(callerDetails.Name(), ".")
		args = append(args, functionKey, nameElements[len(nameElements)-1])
	}
	l.Logger.Debug(msg, args...)
}

func (l *CPILogger) Debugf(format string, args ...any) {
	l.debug(fmt.Sprintf(format, args...))
}

func (l *CPILogger) Infof(format string, args ...any) {
	l.Info(fmt.Sprintf(format, args...))
}

func (l *CPILogger) Warnf(format string, args ...any) {
	l.Warn(fmt.Sprintf(format, args...))
}

func (l *CPILogger) Errorf(format string, args ...any) {
	l.Error(fmt.Sprintf(format, args...))
}

func (l *CPILogger) ChildLogger(component string) LoggerInterface {
	d := l
	d.Logger = l.WithGroup(component)
	return d
}

func (l *CPILogger) WithContext(ctx context.Context) {
	l.Logger = l.With(
		slog.Any(fmt.Sprint(cpiMethodKey), ctx.Value(cpiMethodKey)),
		slog.Any(fmt.Sprint(requestIDKey), ctx.Value(requestIDKey)),
		slog.Any(fmt.Sprint(directorUUIDKey), ctx.Value(directorUUIDKey)),
	)
}
