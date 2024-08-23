package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

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

func printShowEnv() {
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
}

type EnvJSON map[string]string

func envAsJson() EnvJSON {
	values := make(EnvJSON, len(CounterEnv))
	for env, this := range CounterEnv {
		switch that := this.(type) {
		case *bool:
			values[env] = strconv.FormatBool(*that)
		case *string:
			values[env] = strings.Clone(*that)
		case *int64:
			values[env] = strconv.FormatInt(*that, 10)
		default:
			continue
		}
	}
	return values
}
