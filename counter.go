package main

import (
	"crypto/sha512"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

const VERSION = "1.0.0"

var (
	showVersion bool
	counterFile string
	counterName string
	counterDir  string
	doAdd       bool
	doSub       bool
	doDelete    bool
	useForce    bool
	useYes      bool
	neverDelete bool = false
)

func main() {
	handleArguments()
	if showVersion {
		fmt.Println(VERSION)
		os.Exit(0)
	}
	handleEnvironment()

	if err := ensureDir(counterDir, useForce); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if resolved, resolveErr := resolveSymlink(counterFile); resolveErr == nil {
		counterFile = resolved
	}
	if resolved, resolveErr := resolveSymlink(counterDir); resolveErr == nil {
		counterDir = resolved
	}
	counterFile = filepath.Join(counterDir, generateCounterFileName(counterName))
	counter, readErr := readCounter(counterFile)
	if readErr != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", readErr)
		os.Exit(1)
	}

	if doDelete {
		if !useYes {
			_, _ = fmt.Fprintf(os.Stderr, "deleting counter %s (%d) when you re-run with -yes\n", counterName, counter)
			os.Exit(1)
		}
		if neverDelete {
			_, _ = fmt.Fprintf(os.Stderr, "Error: never delete enabled\n")
			os.Exit(1)
		}
		removeErr := os.Remove(counterFile)
		if removeErr != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", removeErr)
		}
		_, _ = fmt.Fprintf(os.Stdout, "counter %s deleted\n", counterName)
		os.Exit(1)

	}

	if !doAdd && !doSub || doAdd && doSub {
		// GET
		fmt.Println(counter)
		os.Exit(0)
	}

	if doAdd {
		counter++
	}

	if doSub {
		counter--
	}

	info, _ := os.Stat(counterFile)
	_ = os.Chmod(counterFile, 0600)

	file, fileErr := os.OpenFile(counterFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0500)
	defer file.Close()
	if fileErr != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", fileErr)
	}
	if writeErr := writeCounter(counterFile, counter, file); writeErr != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", writeErr)
		os.Exit(1)
	}

	_ = os.Chmod(counterFile, info.Mode())

	// Output the final counter value
	fmt.Println(counter)
}

func handleArguments() {
	// Shorthand
	flag.BoolVar(&doAdd, "a", true, "add 1 to the counter")
	flag.BoolVar(&doSub, "s", false, "subtract 1 from the counter")
	flag.BoolVar(&useForce, "F", false, "force overwrite")
	flag.BoolVar(&showVersion, "v", false, "show version")
	flag.StringVar(&counterDir, "d", "/tmp/.counters", "counter directory")
	flag.StringVar(&counterFile, "f", "/tmp/.counters/default", "counter file name")
	flag.StringVar(&counterName, "n", "default", "counter name")

	// Longhand
	flag.BoolVar(&useYes, "yes", false, "your response is yes")
	flag.BoolVar(&doAdd, "add", doAdd, "add 1 to the counter")
	flag.BoolVar(&doSub, "sub", doSub, "subtract 1 from the counter")
	flag.BoolVar(&doSub, "subtract", doSub, "subtract 1 from the counter")
	flag.BoolVar(&useForce, "force", false, "force overwrite")
	flag.BoolVar(&showVersion, "ver", showVersion, "show version")
	flag.BoolVar(&showVersion, "version", showVersion, "show version")
	flag.BoolVar(&doDelete, "delete", false, "remove counter (requires -yes)")
	flag.StringVar(&counterDir, "dir", counterDir, "counter directory")
	flag.StringVar(&counterDir, "directory", counterDir, "counter directory")
	flag.StringVar(&counterName, "name", counterName, "counter name")
	flag.StringVar(&counterFile, "file", counterFile, "counter file name")

	flag.Parse()
}

// handleEnvironment sets properties based on config properties COUNTER_USE_FORCE=1 will set -F | -force on every command
// COUNTER_DIR=path will set -d | -dir on every command, and finally COUNTER_NEVER_DELETE=1 will never delete a counter
// file from the system
func handleEnvironment() {
	envUseForce := os.Getenv("COUNTER_USE_FORCE")
	if envUseForce == "1" {
		useForce = true
	}

	envCounterDir := os.Getenv("COUNTER_DIR")
	if len(envCounterDir) > 0 {
		counterDir = envCounterDir
	}

	envNeverDelete := os.Getenv("COUNTER_NEVER_DELETE")
	if envNeverDelete == "1" {
		neverDelete = true
	}
}

// readCounter reads the counter value from the specified file.
func readCounter(filePath string) (int64, error) {
	counterBytes, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil // If the file doesn't exist, start at 0
		}
		return 0, fmt.Errorf("failed to read counter file: %w", err)
	}
	counterString := strings.TrimSpace(string(counterBytes))
	counter, parseErr := strconv.ParseInt(counterString, 10, 64)
	if parseErr != nil {
		return 0, fmt.Errorf("invalid counter value: %w", parseErr)
	}
	return counter, nil
}

// writeCounter writes the counter value to the specified file.
func writeCounter(filePath string, counter int64, file *os.File) error {
	counterString := strconv.FormatInt(counter, 10)
	bytesWritten, writeErr := file.WriteString(counterString)
	if writeErr != nil {
		return fmt.Errorf("writeCounter.go write error: %w", writeErr)
	}
	if bytesWritten == 0 {
		return fmt.Errorf("only wrote %d of %d bytes to %s", bytesWritten, len(counterString), filePath)
	}
	if err := setImmutable(filePath); err != nil {
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
