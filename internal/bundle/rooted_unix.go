//go:build unix

// SPDX-License-Identifier: MIT
package bundle

import (
	"errors"
	"golang.org/x/sys/unix"
	"os"
)

func lockDescriptor(f *os.File) error {
	err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN) {
		return ErrBundleLocked
	}
	return err
}
func unlockDescriptor(f *os.File) error { return unix.Flock(int(f.Fd()), unix.LOCK_UN) }
