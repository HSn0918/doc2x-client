package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var failureLogMu sync.Mutex

func logFailure(path, traceID, target string, err error) error {
	if path == "" {
		return nil
	}

	if traceID == "" {
		traceID = "unknown"
	}
	timestamp := time.Now().Format(time.RFC3339)
	line := fmt.Sprintf("%s\tlevel=ERROR\ttrace-id=%s\ttarget=%s\tmessage=%v\n", timestamp, traceID, target, err)

	failureLogMu.Lock()
	defer failureLogMu.Unlock()

	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if mkErr := os.MkdirAll(dir, 0o755); mkErr != nil {
			return mkErr
		}
	}

	f, openErr := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if openErr != nil {
		return openErr
	}
	defer f.Close()

	_, writeErr := f.WriteString(line)
	return writeErr
}
