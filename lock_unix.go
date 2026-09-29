//go:build unix

package main

import (
	"os"
	"syscall"
)

// lockFile blocks until it holds an advisory lock on f. Closing f releases it.
func lockFile(f *os.File, exclusive bool) error {
	how := syscall.LOCK_SH
	if exclusive {
		how = syscall.LOCK_EX
	}
	for {
		if err := syscall.Flock(int(f.Fd()), how); err != syscall.EINTR {
			return err
		}
	}
}
