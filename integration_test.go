package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func randomAlphaNumString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, n)
	for i := range result {
		result[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(result)
}

func TestIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "counter")
	countersDir := t.TempDir()

	// Step 1: Build the binary
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}

	export := fmt.Sprintf(`export COUNTER_DIR="%s"`, countersDir)

	// Step 2: Define the test commands and expected outputs
	usageTest := map[string]string{
		export:                                              "",
		binaryPath + " -n test -a":                          "1",
		binaryPath + " -n test -a":                          "2",
		binaryPath + " -n test -S 20":                       "20",
		binaryPath + " -n test -s":                          "19",
		binaryPath + " -n test -R -yes":                     "0",
		binaryPath + " -n test -R":                          "will reset counter test to 0 after you re-run with -yes",
		binaryPath + " -n test -D -yes":                     "counter test deleted",
		binaryPath + " -n daily_hits -cycle daily -in noon": "counter will reset daily at noon",
		binaryPath + " -n daily_hits -a":                    "1",
		binaryPath + " -n daily_hits -s":                    "0",
		binaryPath + " -n daily_hits -rmcc":                 "cycle removed from daily_hits",
		binaryPath + " -v":                                  "2.0.0",
	}

	// Step 3: Run the commands and validate output
	for cmdStr, expectedOutput := range usageTest {
		successKey := randomAlphaNumString(17)
		cmdStr = cmdStr + " || echo " + successKey

		cmd := exec.Command("bash", "-c", cmdStr)
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		if err := cmd.Run(); err != nil {
			t.Errorf("Failed to execute command: %v", err)
		}

		output := out.String()
		if strings.Contains(output, successKey) {
			t.Errorf("Command failed: %s\nOutput:\n%s", cmdStr, output)
		}
		output = strings.TrimSpace(strings.ReplaceAll(output, successKey, ""))
		if len(expectedOutput) == 0 {
			continue
		}
		if !strings.Contains(output, expectedOutput) {
			t.Errorf("Expected output:\n%s\nBut got:\n%s", expectedOutput, output)
		}
	}

	// Step 4: Cleanup
	_ = os.RemoveAll(tmpDir)
}
