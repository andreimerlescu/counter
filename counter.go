package main

import (
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

const VERSION = "1.0.1"

var (
	showVersion   bool
	quantity      int64
	doAdd         bool
	doSub         bool
	doReset       bool
	useForce      bool
	counterFile   string
	counterName   string
	counterDir    string
	doDelete      bool
	useYes        bool
	showEnv       bool
	neverDelete   bool = false
	neverSubtract      = false
	neverReset    bool = false
	neverAdd           = false
	setTo         int64
	neverSetTo    bool = false
)

var CounterEnv = map[string]interface{}{
	"COUNTER_DIR":            &counterDir,
	"COUNTER_QUANTITY":       &quantity,
	"COUNTER_USE_FORCE":      &useForce,
	"COUNTER_NEVER_ADD":      &neverAdd,
	"COUNTER_NEVER_RESET":    &neverReset,
	"COUNTER_NEVER_DELETE":   &neverDelete,
	"COUNTER_NEVER_SET_TO":   &neverSetTo,
	"COUNTER_NEVER_SUBTRACT": &neverSubtract,
}

// handleEnvironment sets properties based on environment variables
func handleEnvironment() {
	for env, this := range CounterEnv {
		thisVal := os.Getenv(env)
		if len(thisVal) == 0 {
			continue
		}
		switch that := this.(type) {
		case *bool:
			*that = thisVal == "1"
		case *string:
			*that = strings.Clone(thisVal)
		case *int64:
			is, err := strconv.ParseInt(thisVal, 10, 64)
			if err == nil {
				*that = is
			} else {
				_, _ = fmt.Fprintf(os.Stderr, "invalid integer value for %s: %s\n", env, thisVal)
				os.Exit(1)
			}
		default:
			continue
		}
	}
}

func main() {
	// Shorthand
	flag.BoolVar(&doAdd, "a", false, "add -q=N (1) to the counter")
	flag.BoolVar(&doSub, "s", false, "subtract -q=N (1) from the counter")
	flag.Int64Var(&setTo, "S", 0, "set counter to value - 0 value ignores this flag")
	flag.BoolVar(&doReset, "R", false, "set counter to 0")
	flag.BoolVar(&doDelete, "D", false, "delete the counter")
	flag.BoolVar(&useForce, "F", false, "force overwrite")
	flag.Int64Var(&quantity, "q", 1, "quantity to either add/subtract from counter")
	flag.BoolVar(&showVersion, "v", false, "show version")
	flag.StringVar(&counterDir, "d", "/tmp/.counters", "counter directory")
	flag.StringVar(&counterFile, "f", "/tmp/.counters/default", "counter file name")
	flag.StringVar(&counterName, "n", "default", "counter name")

	// Longhand
	flag.BoolVar(&useYes, "yes", useYes, "your response is yes")
	flag.BoolVar(&doAdd, "add", doAdd, "add -q=N (1) to the counter")
	flag.BoolVar(&doSub, "sub", doSub, "subtract -q=N (1) from the counter")
	flag.Int64Var(&setTo, "set", setTo, "set counter to value - 0 value ignores this flag")
	flag.BoolVar(&useForce, "force", useForce, "force overwrite")
	flag.BoolVar(&doReset, "reset", doReset, "reset the counter")
	flag.BoolVar(&doDelete, "delete", doDelete, "remove counter (requires -yes)")
	flag.StringVar(&counterDir, "dir", counterDir, "counter directory")
	flag.StringVar(&counterName, "name", counterName, "counter name")
	flag.StringVar(&counterFile, "file", counterFile, "counter file name")

	flag.BoolVar(&showEnv, "env", false, "show environment variables")

	flag.Parse()

	if showVersion {
		fmt.Println(VERSION)
		os.Exit(0)
	}
	handleEnvironment()

	if showEnv {
		for env, this := range CounterEnv {
			switch that := this.(type) {
			case *bool:
				_, _ = fmt.Fprintf(os.Stdout, "%s=%v\n", env, *that)
			case *string:
				_, _ = fmt.Fprintf(os.Stdout, "%s=%v\n", env, *that)
			case *int64:
				_, _ = fmt.Fprintf(os.Stdout, "%s=%v\n", env, *that)
			default:
				continue
			}

		}
		os.Exit(0)
	}

	if len(counterName) == 0 {
		_, _ = fmt.Fprintf(os.Stderr, "Error: counter name is required\n")
		os.Exit(1)
	}

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
		_ = unsetImmutable(counterFile)
		removeErr := os.Remove(counterFile)
		if removeErr != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", removeErr)
		}
		_, _ = fmt.Fprintf(os.Stdout, "counter %s deleted\n", counterName)
		os.Exit(1)
	}

	if doReset && neverReset {
		_, _ = fmt.Fprintf(os.Stderr, "Error: reset operation is disabled by the environment variable\n")
		os.Exit(1)
	}

	if !doReset && !doAdd && !doSub && !doDelete && (setTo == 0 || neverSetTo) {
		fmt.Println(counter)
		os.Exit(0)
	}

	if !doReset && setTo == 0 && doAdd && !neverAdd {
		if x := counter + quantity; x < math.MaxInt64 {
			counter = counter + quantity
		} else {
			counter = math.MaxInt64
		}
	}

	if !doReset && setTo == 0 && doSub && !neverSubtract {
		if x := counter - quantity; x > math.MinInt64 {
			counter = counter - quantity
		} else {
			counter = math.MinInt64
		}
	}

	if !doReset && !neverSetTo && setTo != 0 {
		if setTo < math.MinInt64 {
			counter = math.MinInt64
		} else if setTo > math.MaxInt64 {
			counter = math.MinInt64
		} else {
			counter = setTo
		}
	}

	if doReset {
		if !useYes {
			_, _ = fmt.Fprintf(os.Stderr, "will reset counter %s to 0 after you re-run with -yes\n", counterName)
			os.Exit(1)
		}
		counter = 0
	}

	info, infoErr := os.Stat(counterFile)
	if infoErr == nil {
		_ = os.Chmod(counterFile, 0600)
	}

	file, fileErr := os.OpenFile(counterFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0500)
	defer file.Close()
	if fileErr != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", fileErr)
	}
	if writeErr := writeCounter(counterFile, counter, file); writeErr != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", writeErr)
		os.Exit(1)
	}

	if infoErr == nil {
		_ = os.Chmod(counterFile, info.Mode())
	}

	// Output the final counter value
	fmt.Println(counter)
}

// readCounter reads the counter value from the specified file.
func readCounter(filePath string) (int64, error) {
	counterBytes, err := os.ReadFile(filePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
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
