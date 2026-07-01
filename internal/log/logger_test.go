package log

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNew_TextFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := New(slog.LevelInfo, "text", &buf)
	logger.Info("hello", "key", "value")
	output := buf.String()
	if !strings.Contains(output, "hello") {
		t.Errorf("expected 'hello' in output, got: %s", output)
	}
	if !strings.Contains(output, "key=value") {
		t.Errorf("expected 'key=value' in output, got: %s", output)
	}
}

func TestNew_JSONFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := New(slog.LevelInfo, "json", &buf)
	logger.Info("test-msg", "foo", "bar")
	output := buf.String()
	if !strings.Contains(output, `"msg":"test-msg"`) {
		t.Errorf("expected JSON msg field, got: %s", output)
	}
}

func TestNew_LevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	logger := New(slog.LevelWarn, "text", &buf)
	logger.Info("should-not-appear")
	logger.Warn("should-appear")
	output := buf.String()
	if strings.Contains(output, "should-not-appear") {
		t.Error("info message should be filtered at warn level")
	}
	if !strings.Contains(output, "should-appear") {
		t.Error("warn message should appear at warn level")
	}
}

func TestSetupFileLogging(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "test.log")
	logger, err := SetupFileLogging(logPath, 1024*1024, 3, slog.LevelInfo)
	if err != nil {
		t.Fatalf("SetupFileLogging failed: %v", err)
	}
	logger.Info("file-log-test")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}
	if !strings.Contains(string(data), "file-log-test") {
		t.Errorf("expected log content in file, got: %s", string(data))
	}
}

func TestRotateIfNeeded(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "rotate.log")
	os.WriteFile(logPath, bytes.Repeat([]byte("x"), 100), 0600)
	err := rotateIfNeeded(logPath, 50, 3)
	if err != nil {
		t.Fatalf("rotateIfNeeded failed: %v", err)
	}
	if _, err := os.Stat(logPath + ".1"); os.IsNotExist(err) {
		t.Error("expected rotated file .1 to exist")
	}
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Error("expected original file to be renamed")
	}
}
