package main

import (
	"errors"
	"os"
	"strings"
)

const (
	exitSuccess          = 0
	exitUserError        = 1
	exitIntegrityFailure = 2
	exitVerifyFailure    = 3
	exitSystemError      = 3
)

func classifyBundleLoadError(err error) int {
	switch {
	case errors.Is(err, os.ErrNotExist):
		return exitUserError
	case errors.Is(err, os.ErrPermission):
		return exitSystemError
	}

	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "unmarshal") || strings.Contains(msg, "scan") {
		return exitIntegrityFailure
	}
	return exitSystemError
}
