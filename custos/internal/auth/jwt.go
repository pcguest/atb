// SPDX-License-Identifier: MIT
package auth

import atbauth "github.com/pcguest/atb/pkg/auth"

// JWTValidator aliases the shared ATB/Custos OIDC validator.
type JWTValidator = atbauth.JWTValidator

// NewJWTValidator creates a shared OIDC JWT validator.
var NewJWTValidator = atbauth.NewJWTValidator
