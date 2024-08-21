package main

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// TestGenerateCounterFileName tests the generateCounterFileName function
func TestGenerateCounterFileName(t *testing.T) {
	expected := ".named.893e3b29586bf2531a893d15.counter"
	name := "myCounter"
	result := generateCounterFileName(name)
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

// TestEnsureDir tests the ensureDir function
func TestEnsureDir(t *testing.T) {
	tmpDir := t.TempDir() // Create a temporary directory for testing
	testDir := filepath.Join(tmpDir, "testDir")
	if err := ensureDir(testDir, false); err == nil {
		t.Errorf("Expected error for non-existent directory without force flag, got nil")
	}
	if err := ensureDir(testDir, true); err != nil {
		t.Errorf("Failed to create directory with force flag: %v", err)
	}
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		t.Errorf("Directory was not created")
	}
}

// TestReadCounter tests the readCounter function
func TestReadCounter(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "counterFile")
	counter, err := readCounter(testFile)
	if err != nil {
		t.Errorf("Expected no error for non-existent file, got: %v", err)
	}
	if counter != 0 {
		t.Errorf("Expected counter to be 0 for non-existent file, got %d", counter)
	}
	expectedCounter := int64(42)
	tmpFile, fileErr := os.OpenFile(testFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	defer tmpFile.Close()
	if fileErr != nil {
		t.Fatalf("failed to open file: %v", fileErr)
	}
	if err := writeCounter(testFile, expectedCounter, tmpFile); err != nil {
		t.Errorf("Failed to write counter: %v", err)
	}
	counter, err = readCounter(testFile)
	if err != nil {
		t.Errorf("Failed to read counter: %v", err)
	}
	if counter != expectedCounter {
		t.Errorf("Expected %d, got %d", expectedCounter, counter)
	}
}

// TestWriteCounter tests the writeCounter function
func TestWriteCounter(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "counterFile")
	expectedCounter := int64(123)
	tmpFile, fileErr := os.OpenFile(testFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	defer tmpFile.Close()
	if fileErr != nil {
		t.Fatalf("failed to open file: %v", fileErr)
	}
	if err := writeCounter(testFile, expectedCounter, tmpFile); err != nil {
		t.Errorf("Failed to write counter: %v", err)
	}
	counter, err := readCounter(testFile)
	if err != nil {
		t.Errorf("Failed to read counter: %v", err)
	}
	if counter != expectedCounter {
		t.Errorf("Expected %d, got %d", expectedCounter, counter)
	}
}

// TestSetUnsetImmutable tests the setImmutable and unsetImmutable functions
func TestSetUnsetImmutable(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "counterFile")
	expectedCounter := int64(456)
	tmpFile, fileErr := os.OpenFile(testFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	defer tmpFile.Close()
	if fileErr != nil {
		t.Fatalf("failed to open file: %v", fileErr)
	}
	if err := writeCounter(testFile, expectedCounter, tmpFile); err != nil {
		t.Errorf("Failed to write counter: %v", err)
	}
	if err := setImmutable(testFile); err != nil {
		t.Errorf("Failed to set file immutable: %v", err)
	}
	if err := unsetImmutable(testFile); err != nil {
		t.Errorf("Failed to unset file immutable: %v", err)
	}
}

// BenchmarkWriteCounter benchmarks the writeCounter function.
func BenchmarkWriteCounter(b *testing.B) {
	dir, err := os.MkdirTemp(b.TempDir(), "bwc-"+strconv.Itoa(b.N))
	if err != nil {
		b.Fatalf("failed to create benchmark directory: %v", err)
	}
	defer os.RemoveAll(dir)
	filePath := filepath.Join(dir, "counter_test_file"+strconv.Itoa(b.N))

	tmpFile, fileErr := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if fileErr != nil {
		b.Fatalf("failed to open file: %v", fileErr)
	}
	defer tmpFile.Close()

	for i := 0; i < b.N; i++ {
		counter := int64(i)
		err := writeCounter(filePath, counter, tmpFile)
		if err != nil {
			b.Fatalf("failed to write counter: %v", err)
		}
	}
}

// BenchmarkReadCounter benchmarks the readCounter function.
func BenchmarkReadCounter(b *testing.B) {
	dir, err := os.MkdirTemp(b.TempDir(), "BenchmarkReadCounter"+strconv.Itoa(b.N))
	if err != nil {
		b.Fatalf("failed to create benchmark directory: %v", err)
	}
	defer os.RemoveAll(dir)

	filePath := filepath.Join(dir, "counter_test_file"+strconv.Itoa(b.N))
	initialCounter := int64(12345)
	tmpFile, fileErr := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	defer tmpFile.Close()
	if fileErr != nil {
		b.Fatalf("failed to open file: %v", fileErr)
	}
	err = writeCounter(filePath, initialCounter, tmpFile)
	if err != nil {
		b.Fatalf("failed to write initial counter: %v", err)
	}

	for i := 0; i < b.N; i++ {
		_, err := readCounter(filePath)
		if err != nil {
			b.Fatalf("failed to read counter: %v", err)
		}
	}
}

// BenchmarkGenerateCounterFileName benchmarks the generateCounterFileName function.
func BenchmarkGenerateCounterFileName(b *testing.B) {
	for i := 0; i < b.N; i++ {
		generateCounterFileName("BenchmarkGenerateCounterFileName")
	}
}

// BenchmarkEnsureDir benchmarks the ensureDir function.
func BenchmarkEnsureDir(b *testing.B) {
	dir := filepath.Join(os.TempDir(), "BenchmarkEnsureDir"+strconv.Itoa(b.N))

	for i := 0; i < b.N; i++ {
		err := ensureDir(dir, true)
		if err != nil {
			b.Fatalf("failed to ensure directory: %v", err)
		}
		_ = os.RemoveAll(dir)
	}
}

// BenchmarkResolveSymlink benchmarks the resolveSymlink function.
func BenchmarkResolveSymlink(b *testing.B) {
	dir := filepath.Join(os.TempDir(), "BenchmarkResolveSymlink"+strconv.Itoa(b.N))
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		b.Fatalf("failed to create benchmark directory: %v", err)
	}
	defer os.RemoveAll(dir)

	filePath := filepath.Join(dir, "symlink_test_file")
	_ = os.WriteFile(filePath, []byte("test"), 0666)

	symlinkPath := filePath + "_symlink"
	_ = os.Symlink(filePath, symlinkPath)

	for i := 0; i < b.N; i++ {
		_, err := resolveSymlink(symlinkPath)
		if err != nil {
			b.Fatalf("failed to resolve symlink: %v", err)
		}
	}
}
