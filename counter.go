package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Counter struct {
	Value       int64     `json:"value"`
	Path        string    `json:"path"`
	CreatedAt   time.Time `json:"created_at"`
	Cycle       string    `json:"cycle"`
	CycleIn     string    `json:"cycle_in"`
	DeleteAfter time.Time `json:"delete_after"`
}

type versionData struct {
    Version string `json:"version"`
}

var Version = versionData{VERSION}


func main() {
	initArgs()

	if showVersion {
		if showJson {
			jsonData, jsonErr := json.Marshal(Version)
			if jsonErr == nil {
				fmt.Println(string(jsonData))
				os.Exit(0)
			} else {
				_, _ = fmt.Fprintf(os.Stderr, "JSON Error: %s\n", jsonErr.Error())
			}
		}
		fmt.Println(VERSION)
		os.Exit(0)
	}

	if showUsage {
		printUsage()
		os.Exit(0)
	}

	handleEnvironment()

	if showEnv {
		if showJson {
			data := envAsJson()
			jsonData, jsonErr := json.Marshal(data)
			if jsonErr == nil {
				fmt.Println(string(jsonData))
				os.Exit(0)
			} else {
				_, _ = fmt.Fprintf(os.Stderr, "JSON Error: %s\n", jsonErr.Error())
			}
		}
		printShowEnv()
		os.Exit(0)
	}

	if strings.EqualFold(counterFile, DefaultCounterFile) && strings.EqualFold(counterName, DefaultCounterName) {
		_, _ = fmt.Fprintf(os.Stderr, "Error: -name or -file is required\n")
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
	if counterFile == DefaultCounterFile {
		if counterName == DefaultCounterName {
			_, _ = fmt.Fprintf(os.Stderr, "Error: counter name is required\n")
			os.Exit(1)
		}
		counterFile = filepath.Join(counterDir, generateCounterFileName(counterName))
	} else {
		if counterName == "" {
			s := strings.Clone(counterFile)
			if s[0] != '/' {
				counterFile = filepath.Join(counterDir, counterFile)
			}
		} else {
			counterFile = filepath.Join(counterDir, counterName)
		}
	}
	counter, readErr := readCounter(counterFile)
	if readErr != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", readErr)
		os.Exit(1)
	}

	if cycle != "" {
		shouldReset, err := shouldResetCounter(counterFile, cycle, cycleIn)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if shouldReset {
			counter.Value = 0
        } else {
            _, _ = fmt.Fprintf(os.Stderr, "Failed to reset the counter: %#v\nshouldReset: %#v",
                counter, shouldReset)
            os.Exit(1)
        }
	}

	if doDelete {
		if neverDelete {
			_, _ = fmt.Fprintf(os.Stderr, "Error: never delete enabled\n")
			os.Exit(1)
		}
		if !useYes {
			_, _ = fmt.Fprintf(os.Stderr, "deleting counter %s (%d) when you re-run with -yes\n", counterName, counter.Value)
			os.Exit(1)
		}
		_ = unsetImmutable(counterFile)
		removeErr := os.Remove(counterFile)
		if removeErr == nil {
			_, _ = fmt.Fprintf(os.Stdout, "counter %s deleted\n", counterName)
		}
		os.Exit(1)
	}

	if cycle != DefaultCycle {
		if neverCycle {
			_, _ = fmt.Fprintf(os.Stderr, "Error: env COUNTER_NEVER_CYCLE prevented cycle\n")
			os.Exit(1)
		}
		targetTime, err := parseCycleIn(cycle, cycleIn)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		info, err := os.Stat(counterFile)
		if err == nil {
			if info.ModTime().Before(targetTime) {
				counter.Value = 0
			}
		}

		// Update the counter with the cycle and cycleIn information
		counter.Cycle = strings.Clone(cycle)
		counter.CycleIn = strings.Clone(cycleIn)
	}

	if doReset && neverReset {
		_, _ = fmt.Fprintf(os.Stderr, "Error: reset operation is disabled by the environment variable\n")
		os.Exit(1)
	}

	if cycle == DefaultCycle || cycleIn == DefaultCycleIn || doAdd || doSub {
		if !doReset && !doAdd && !doSub && !doDelete && (setTo == 0 || neverSetTo) {
			if showJson {
				outputJson(counter)
			} else {
				fmt.Println(counter.Value)
			}
			os.Exit(0)
		}
	}

	if !doReset && setTo == 0 && doAdd && !neverAdd {
		if x := counter.Value + quantity; x < math.MaxInt64 {
			counter.Value = counter.Value + quantity
		} else {
			counter.Value = math.MaxInt64
		}
	}

	if !doReset && setTo == 0 && doSub && !neverSubtract {
		if x := counter.Value - quantity; x > math.MinInt64 {
			counter.Value = counter.Value - quantity
		} else {
			counter.Value = math.MinInt64
		}
	}

	if !doReset && !neverSetTo && setTo != 0 {
		if setTo < math.MinInt64 {
			counter.Value = math.MinInt64
		} else if setTo > math.MaxInt64 {
			counter.Value = math.MinInt64
		} else {
			counter.Value = setTo
		}
	}

	if doReset {
		if !useYes {
			_, _ = fmt.Fprintf(os.Stderr, "will reset counter %s to 0 after you re-run with -yes\n", counterName)
			os.Exit(1)
		}
		counter.Value = 0
	}

	info, infoErr := os.Stat(counterFile)
	if infoErr == nil {
		_ = os.Chmod(counterFile, 0600)
	}

	file, fileErr := os.OpenFile(counterFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0500)
	defer func() {
		_ = file.Close()
	}()
	if fileErr != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", fileErr)
	}
	if writeErr := writeCounter(counter, file); writeErr != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", writeErr)
		os.Exit(1)
	}

	if infoErr == nil {
		_ = os.Chmod(counterFile, info.Mode())
	}

	if cycle != DefaultCycle && cycleIn != DefaultCycleIn && !doAdd && !doSub {
		fmt.Printf("counter %s will reset %s at %s\n", counterName, counter.Cycle, counter.CycleIn)
	} else {
		if showJson {
			outputJson(counter)
		} else {
			fmt.Println(counter.Value)
		}
	}

	os.Exit(0)
}
