//go:build tools

package tools

import (
	// Keep gosec pinned in go.mod for reproducible local security scans.
	_ "github.com/securego/gosec/v2/cmd/gosec"
)
