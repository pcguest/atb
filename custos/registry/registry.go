// SPDX-License-Identifier: MIT
// Package registry defines the Custos known-tool registry scaffold.
package registry

import "github.com/pcguest/custos/discovery"

// ToolSig is the registry view of a discovered AI tool signature.
type ToolSig = discovery.ToolSig

// Registry stores known AI tool signatures.
type Registry struct {
	known []ToolSig
}

// Lookup returns a known tool signature by name.
func (r *Registry) Lookup(name string) (*ToolSig, bool) {
	// TODO: implement registry lookup semantics.
	_ = r
	_ = name
	return nil, false
}

// Register adds a known tool signature.
func (r *Registry) Register(sig ToolSig) {
	// TODO: implement registry update semantics.
	_ = r
	_ = sig
}
