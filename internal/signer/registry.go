// SPDX-License-Identifier: MIT

package signer

import (
	"context"
	"fmt"
)

// BackendFactory constructs a Signer for a backend-scoped key identifier.
type BackendFactory func(ctx context.Context, keyID string) (Signer, error)

var backends = map[string]BackendFactory{}

// Register makes a backend factory available to the CLI. It is intended for
// build-tagged backend packages that register themselves from init.
func Register(name string, f BackendFactory) {
	backends[name] = f
}

// Resolve returns a registered backend signer.
func Resolve(ctx context.Context, name, keyID string) (Signer, error) {
	f, ok := backends[name]
	if !ok {
		return nil, fmt.Errorf("signer: unknown backend %q (was the binary built with -tags %s?)", name, name)
	}
	return f(ctx, keyID)
}
