package log

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
)

func New(level slog.Level, format string, w io.Writer) *slog.Logger {
	opts := &slog.HandlerOptions{Level: level}
	var handler slog.Handler
	if format == "json" {
		handler = slog.NewJSONHandler(w, opts)
	} else {
		handler = slog.NewTextHandler(w, opts)
	}
	return slog.New(handler)
}

func SetupFileLogging(logPath string, maxBytes int64, maxFiles int, level slog.Level) (*slog.Logger, error) {
	dir := filepath.Dir(logPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("log: create dir %s: %w", dir, err)
	}
	if err := rotateIfNeeded(logPath, maxBytes, maxFiles); err != nil {
		return nil, fmt.Errorf("log: rotate: %w", err)
	}
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("log: open %s: %w", logPath, err)
	}
	return New(level, "json", f), nil
}

func rotateIfNeeded(logPath string, maxBytes int64, maxFiles int) error {
	info, err := os.Stat(logPath)
	if err != nil {
		return nil
	}
	if info.Size() < maxBytes {
		return nil
	}
	for i := maxFiles; i >= 1; i-- {
		old := fmt.Sprintf("%s.%d", logPath, i)
		newer := logPath
		if i > 1 {
			newer = fmt.Sprintf("%s.%d", logPath, i-1)
		}
		if i == maxFiles {
			os.Remove(old)
		}
		if _, err := os.Stat(newer); err == nil {
			os.Rename(newer, old)
		}
	}
	return nil
}

func CleanOldLogs(logPath string, maxFiles int) error {
	dir := filepath.Dir(logPath)
	base := filepath.Base(logPath)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	var logFiles []string
	for _, e := range entries {
		if matched, _ := filepath.Match(base+".*", e.Name()); matched {
			logFiles = append(logFiles, filepath.Join(dir, e.Name()))
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(logFiles)))
	for i := maxFiles; i < len(logFiles); i++ {
		os.Remove(logFiles[i])
	}
	return nil
}
