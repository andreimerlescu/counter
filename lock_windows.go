//go:build windows

package main

import (
	"os"

	"golang.org/x/sys/windows"
)

// lockFile blocks until it holds a lock on the whole of f. Closing f releases it.
func lockFile(f *os.File, exclusive bool) error {
	var flags uint32
	if exclusive {
		flags = windows.LOCKFILE_EXCLUSIVE_LOCK
	}
	return windows.LockFileEx(windows.Handle(f.Fd()), flags, 0, ^uint32(0), ^uint32(0), new(windows.Overlapped))
}
