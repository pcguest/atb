// SPDX-License-Identifier: MIT
package auth

import atbauth "github.com/pcguest/atb/pkg/auth"

// Role aliases the shared ATB/Custos RBAC role type. Custos keeps this internal
// package for daemon-specific middleware, but role semantics live in pkg/auth.
type Role = atbauth.Role

const (
	RoleViewer   = atbauth.RoleViewer
	RoleAuditor  = atbauth.RoleAuditor
	RoleOperator = atbauth.RoleOperator
	RoleAdmin    = atbauth.RoleAdmin
)
