// SPDX-License-Identifier: MIT
// Package discovery defines the Custos tool-discovery boundary.
package discovery

// ToolSig describes one AI tool or endpoint observed during discovery.
type ToolSig struct {
	Name     string
	Vendor   string
	Kind     string
	Endpoint string
}

// Discoverer scans an authorised environment for AI tool signatures.
type Discoverer interface {
	// TODO: define scan sources, permissions, and result freshness semantics.
	Scan() ([]ToolSig, error)
}
