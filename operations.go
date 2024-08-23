package main

import (
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// shouldResetCounter looks at the counterFile and cycle to signal
func shouldResetCounter(counterFile, cycle, cycleIn string) (bool, error) {
	info, err := os.Stat(counterFile)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return true, nil
		}
		return false, err
	}
	lastModTime := info.ModTime()
	now := time.Now()
	switch cycle {
	case "hourly":
		return now.Sub(lastModTime).Hours() >= 1, nil
	case "daily":
		return now.Sub(lastModTime).Hours() >= 24, nil
	case "weekly":
		return now.Sub(lastModTime).Hours() >= 168, nil
	case "monthly":
		return now.Sub(lastModTime).Hours() >= 720, nil
	case "annually":
		return now.Sub(lastModTime).Hours() >= 8760, nil
	default:
		return false, fmt.Errorf("unsupported cycle: %s", cycle)
	}
}

func readCounter(filePath string) (Counter, error) {
	var counter Counter

	counterBytes, err := os.ReadFile(filePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Counter{Value: 0, Path: filePath, CreatedAt: time.Now()}, nil
		}
		return Counter{}, fmt.Errorf("failed to read counter file: %w", err)
	}

	err = json.Unmarshal(counterBytes, &counter)
	if err != nil {
		// Handle legacy integer-only counter files
		value, parseErr := strconv.ParseInt(strings.TrimSpace(string(counterBytes)), 10, 64)
		if parseErr != nil {
			return Counter{}, fmt.Errorf("invalid counter value: %w", parseErr)
		}
		counter.Value = value
		counter.Path = filePath
		counter.CreatedAt = time.Now()
	}

	return counter, nil
}

func writeCounter(counter Counter, file *os.File) error {
	counterBytes, err := json.MarshalIndent(counter, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal counter to JSON: %w", err)
	}

	bytesWritten, writeErr := file.Write(counterBytes)
	if writeErr != nil {
		return fmt.Errorf("writeCounter.go write error: %w", writeErr)
	}
	if bytesWritten == 0 {
		return fmt.Errorf("only wrote %d bytes to %s", bytesWritten, counter.Path)
	}

	// Set the file as immutable (if applicable)
	if err := setImmutable(counter.Path); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Warning: Could not set the file as immutable: %v\n", err)
	}
	return nil
}

// resolveSymlink resolves a symlink to its actual path.
func resolveSymlink(path string) (string, error) {
	return filepath.EvalSymlinks(path)
}

// ensureDir ensures that a directory exists.
func ensureDir(dir string, force bool) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if force {
			return os.MkdirAll(dir, 0755)
		}
		return fmt.Errorf("directory %s does not exist", dir)
	}
	return nil
}

// generateCounterFileName generates a counter file name using SHA-512 hashing and some magick
func generateCounterFileName(name string) string {
	hash := sha512.Sum512([]byte(name))
	x := hex.EncodeToString(hash[:])
	y := x[96:99] + x[39:45] + x[63:69] + x[93:99] + x[69:72]
	return fmt.Sprintf(".named.%s.counter", y)
}

// setImmutable sets the immutable flag on a file.
func setImmutable(filePath string) error {
	return syscall.Chmod(filePath, syscall.S_IRUSR|syscall.S_IRGRP|syscall.S_IROTH) // Set to read-only (as an alternative to immutable)
}

// unsetImmutable removes the immutable flag from a file.
func unsetImmutable(filePath string) error {
	return syscall.Chmod(filePath, syscall.S_IRUSR|syscall.S_IWUSR|syscall.S_IRGRP|syscall.S_IROTH) // Set to writable
}

func outputJson(counter Counter) {
	jsonData, err := json.MarshalIndent(counter, "", "  ")
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: could not marshal counter to JSON: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(jsonData))
}
