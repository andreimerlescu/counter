package main

import (
	"bytes"
	"errors"
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

	// Step 1: Build the binary
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")

	err := cmd.Run()
	if err != nil {
		t.Fatalf("cmd.Run() failed with %s\n", err)
	}

	// Step 2: Define the test commands and expected outputs
	usageTest := map[string]string{
		binaryPath + " -n test1 -a":                         "1",
		binaryPath + " -n test -S 20":                       "20",
		binaryPath + " -n test -s":                          "19",
		binaryPath + " -n test -R -yes":                     "0",
		binaryPath + " -n test -R":                          "will reset counter test to 0 after you re-run with -yes",
		binaryPath + " -n test -D -yes":                     "",
		binaryPath + " -n daily_hits -cycle daily -in noon": "counter daily_hits will reset daily at noon",
		binaryPath + " -n daily_hits -a":                    "1",
		binaryPath + " -n daily_hits -s":                    "-1",
		binaryPath + " -n daily_hits -rmcc":                 "cycle removed from daily_hits",
		binaryPath + " -v":                                  "2.0.0",
	}

	// Step 3: Run the commands and validate output
	for cmdStr, expectedOutput := range usageTest {
		t.Run(cmdStr, func(t *testing.T) {

			successKey := randomAlphaNumString(17)
			cmdStr = cmdStr + " && echo " + successKey

			cmd := exec.Command("bash", "-c", cmdStr)

			newEnv := append(cmd.Environ(),
				fmt.Sprintf("%s=%s", "COUNTER_DIR", tmpDir),
				fmt.Sprintf("%s=%s", "COUNTER_USE_FORCE", "1"))
			cmd.Env = newEnv
			cmd.Dir = tmpDir

			var out bytes.Buffer
			cmd.Stdout = &out
			cmd.Stderr = &out
			err := cmd.Run()
			output := out.String()
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) && !strings.Contains(output, successKey) && !strings.Contains(output, expectedOutput) {
				t.Fatalf("FAILURE\nTest: %s\n: %v\nOutput:\n%s\nExpected Output:\n%s\n", cmdStr, err, output, expectedOutput)
			}
		})
	}

	// Step 4: Cleanup
	_ = os.RemoveAll(tmpDir)
}
