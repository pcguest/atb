// SPDX-License-Identifier: MIT
// Package proxy exposes the supported HTTPS capture proxy API.
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
	LoggingHandler = internalproxy.LoggingHandler
	// Deprecated: use LoggingHandler.
	StubHandler = internalproxy.StubHandler
	Proxy       = internalproxy.Proxy
	Runner      = internalproxy.Runner
)

var ErrInvalidConfig = internalproxy.ErrInvalidConfig

var (
	NewProxy           = internalproxy.NewProxy
	DefaultTargetHosts = internalproxy.DefaultTargetHosts
)
