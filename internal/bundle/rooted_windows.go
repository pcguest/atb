//go:build windows

// SPDX-License-Identifier: MIT
package bundle

import "os"

func lockDescriptor(f *os.File) error   { return lockFileEx(f) }
func unlockDescriptor(f *os.File) error { return unlockFileEx(f) }
