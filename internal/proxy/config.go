// SPDX-License-Identifier: MIT
package proxy

import (
	"errors"
	"fmt"
	"strings"

	"github.com/pcguest/atb/internal/identity"
)

var (
	// ErrInvalidConfig indicates ProxyConfig failed validation.
	ErrInvalidConfig = errors.New("proxy: invalid configuration")
)

// ProxyConfig configures the local HTTPS capture proxy.
type ProxyConfig struct {
	ListenAddr     string
	TargetHosts    []string
	BundlePath     string
	IdentityMap    map[string]string
	Identity       identity.Resolver
	CustosEndpoint string // New field
	// CustosToken authenticates auto-pushes to the Custos ingest endpoint as
	// an Authorization: Bearer header. Empty means unauthenticated (dev mode).
	CustosToken string
	// CaptureBodies retains raw request/response bodies in recorded events.
	// Default false: only a SHA-256 digest and byte length are recorded, so an
	// always-on recorder does not persist prompts, completions, or PII. Enable
	// only in environments that accept storing raw content in the bundle.
	CaptureBodies bool
}

// Validate checks required fields and normalises host entries.
func (c *ProxyConfig) Validate() error {
	if c == nil {
		return fmt.Errorf("%w: nil config", ErrInvalidConfig)
	}
	if strings.TrimSpace(c.ListenAddr) == "" {
		return fmt.Errorf("%w: listen address is required", ErrInvalidConfig)
	}
	if strings.TrimSpace(c.BundlePath) == "" {
		return fmt.Errorf("%w: bundle path is required", ErrInvalidConfig)
	}
	if len(c.TargetHosts) == 0 {
		return fmt.Errorf("%w: at least one target host is required", ErrInvalidConfig)
	}
	normalised := make([]string, 0, len(c.TargetHosts))
	for _, host := range c.TargetHosts {
		host = strings.TrimSpace(strings.ToLower(host))
		if host == "" {
			continue
		}
		normalised = append(normalised, host)
	}
	if len(normalised) == 0 {
		return fmt.Errorf("%w: at least one target host is required", ErrInvalidConfig)
	}
	c.TargetHosts = normalised
	if c.IdentityMap == nil {
		c.IdentityMap = map[string]string{}
	}
	return nil
}

// ResolveIdentity returns the display name for an API key when configured.
func (c ProxyConfig) ResolveIdentity(apiKey string) identity.Identity {
	if id, ok := c.resolveIdentity(apiKey); ok {
		return id
	}
	return identity.Identity{DisplayName: identity.FallbackDisplayName(apiKey)}
}

func (c ProxyConfig) resolveIdentity(apiKey string) (identity.Identity, bool) {
	if apiKey == "" {
		return identity.Identity{}, false
	}
	if c.Identity != nil {
		id, err := c.Identity.Resolve(apiKey)
		if err == nil {
			return id, true
		}
	}
	if len(c.IdentityMap) > 0 {
		if name, ok := c.IdentityMap[apiKey]; ok {
			return identity.Identity{DisplayName: strings.TrimSpace(name)}, true
		}
	}
	return identity.Identity{}, false
}
