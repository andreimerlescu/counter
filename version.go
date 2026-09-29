package main

import (
	_ "embed"
	"strings"
)

// versionFile holds the repository VERSION file, embedded at build time.
//
//go:embed VERSION
var versionFile string

// BinaryVersion returns the embedded VERSION, or v0.0.0 if it is empty.
// It is a pure function of an immutable string and safe for concurrent use.
func BinaryVersion() string {
	if v := strings.TrimSpace(versionFile); v != "" {
		return v
	}
	return "v0.0.0"
}
