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
	// ErrBodyTooLarge indicates a request or response body exceeded MaxBodyBytes.
	ErrBodyTooLarge = errors.New("proxy: body exceeds MaxBodyBytes")
)

// BodyTooLargeError reports the configured limit and the number of bytes
// observed before the proxy rejected a body. Observed is a lower bound for
// streamed bodies because the reader stops at limit+1.
type BodyTooLargeError struct {
	Limit    int64
	Observed int64
}

func (e *BodyTooLargeError) Error() string {
	return fmt.Sprintf("%v (limit %d bytes, observed at least %d bytes)", ErrBodyTooLarge, e.Limit, e.Observed)
}

// Unwrap lets callers classify the error with errors.Is.
func (e *BodyTooLargeError) Unwrap() error { return ErrBodyTooLarge }

const (
	// DefaultMaxBodyBytes is the default per-message body cap for capture and
	// in-memory buffering (32 MiB). Large multimodal payloads need an explicit
	// --max-body-bytes increase; unbounded buffering is never the default.
	DefaultMaxBodyBytes int64 = 32 << 20
	// MaxBodyBytesLimit is the largest per-message cap accepted from the CLI.
	// Keeping the value comfortably below MaxInt64 prevents limit arithmetic
	// from overflowing and bounds the blast radius of an accidental setting.
	MaxBodyBytesLimit int64 = 256 << 20
	// DefaultMaxInFlightBodyBytes bounds aggregate request/response buffering.
	// Each exchange reserves twice its per-message cap because a request body
	// can remain live while its response is read.
	DefaultMaxInFlightBodyBytes int64 = 512 << 20
)

// ProxyConfig configures the local HTTPS capture proxy.
type ProxyConfig struct {
	ListenAddr      string
	TargetHosts     []string
	BundlePath      string
	IdentityMap     map[string]string
	Identity        identity.Resolver
	MortiseEndpoint string
	// MortiseToken authenticates auto-pushes to the Mortise ingest endpoint as
	// an Authorization: Bearer header. Empty means unauthenticated (dev mode).
	MortiseToken string
	// CaptureBodies retains raw request/response bodies in recorded events.
	// Default false: only a SHA-256 digest and byte length are recorded, so an
	// always-on recorder does not persist prompts, completions, or PII. Enable
	// only in environments that accept storing raw content in the bundle.
	CaptureBodies bool
	// MaxBodyBytes caps how much of each request/response body is read into
	// memory for capture and forwarding. Zero means DefaultMaxBodyBytes.
	// Negative values are invalid.
	MaxBodyBytes int64
}

// Validate checks required fields without mutating the configuration.
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
	normalised := normalisedTargetHosts(c.TargetHosts)
	if len(normalised) == 0 {
		return fmt.Errorf("%w: at least one target host is required", ErrInvalidConfig)
	}
	if c.MaxBodyBytes < 0 || c.MaxBodyBytes > MaxBodyBytesLimit {
		return fmt.Errorf("%w: MaxBodyBytes must be between 0 and %d", ErrInvalidConfig, MaxBodyBytesLimit)
	}
	return nil
}

// EffectiveMaxBodyBytes returns the configured body cap, applying the default
// when MaxBodyBytes is zero.
func (c ProxyConfig) EffectiveMaxBodyBytes() int64 {
	if c.MaxBodyBytes == 0 {
		return DefaultMaxBodyBytes
	}
	return c.MaxBodyBytes
}

func (c ProxyConfig) normalised() ProxyConfig {
	c.TargetHosts = normalisedTargetHosts(c.TargetHosts)
	if c.IdentityMap == nil {
		c.IdentityMap = map[string]string{}
	}
	if c.MaxBodyBytes == 0 {
		c.MaxBodyBytes = DefaultMaxBodyBytes
	}
	return c
}

func normalisedTargetHosts(hosts []string) []string {
	normalised := make([]string, 0, len(hosts))
	for _, host := range hosts {
		host = strings.TrimSpace(strings.ToLower(host))
		if host == "" {
			continue
		}
		normalised = append(normalised, host)
	}
	return normalised
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
