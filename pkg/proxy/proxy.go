// SPDX-License-Identifier: MIT
// Package proxy re-exports the internal HTTPS capture proxy scaffold.
package proxy

import (
	internalproxy "github.com/pcguest/atb/internal/proxy"
)

const (
	TypeLLMRequest  = internalproxy.TypeLLMRequest
	TypeLLMResponse = internalproxy.TypeLLMResponse
)

type (
	ProxyConfig    = internalproxy.ProxyConfig
	RequestRecord  = internalproxy.RequestRecord
	ResponseRecord = internalproxy.ResponseRecord
	Handler        = internalproxy.Handler
	StubHandler    = internalproxy.StubHandler
	Proxy          = internalproxy.Proxy
	Runner         = internalproxy.Runner
)

var ErrInvalidConfig = internalproxy.ErrInvalidConfig

var (
	NewProxy           = internalproxy.NewProxy
	DefaultTargetHosts = internalproxy.DefaultTargetHosts
)
