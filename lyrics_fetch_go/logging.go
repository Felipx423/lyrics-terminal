package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	logFileName = "lyrics.log"
	logMaxSize  = 5 * 1024 * 1024
	logMaxFiles = 5
)

var logMu sync.Mutex

func logPath() string {
	return filepath.Join(cacheDir, logFileName)
}

func logEvent(label string, value any) {
	logMu.Lock()
	defer logMu.Unlock()
	if err := appendLogLine(label, value); err != nil {
		return
	}
}

func appendLogLine(label string, value any) error {
	if err := ensureDir(cacheDir); err != nil {
		return err
	}
	path := logPath()
	if err := rotateLogs(path); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	line := fmt.Sprintf("%s %s", time.Now().Format(time.RFC3339), label)
	if value != nil {
		line += fmt.Sprintf(": %v", value)
	}
	line += "\n"
	_, err = f.WriteString(line)
	return err
}

func rotateLogs(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return nil
	}
	if info.Size() < logMaxSize {
		return nil
	}
	oldest := fmt.Sprintf("%s.%d", path, logMaxFiles-1)
	_ = os.Remove(oldest)
	for i := logMaxFiles - 2; i >= 1; i-- {
		src := fmt.Sprintf("%s.%d", path, i)
		dst := fmt.Sprintf("%s.%d", path, i+1)
		if _, err := os.Stat(src); err == nil {
			_ = os.Rename(src, dst)
		}
	}
	_ = os.Remove(fmt.Sprintf("%s.1", path))
	return os.Rename(path, fmt.Sprintf("%s.1", path))
}
