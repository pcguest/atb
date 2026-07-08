// SPDX-License-Identifier: MIT
package auth

// Role represents a user's role within local ATB/Mortise HTTP APIs.
type Role string

const (
	// RoleViewer has read-only access to view data.
	RoleViewer Role = "viewer"
	// RoleAuditor has read-only access to view and generate reports.
	RoleAuditor Role = "auditor"
	// RoleOperator has read-write access to manage data and configurations.
	RoleOperator Role = "operator"
	// RoleAdmin has full administrative access.
	RoleAdmin Role = "admin"
)

// IsValid checks if the role is one of the predefined roles.
func (r Role) IsValid() bool {
	switch r {
	case RoleViewer, RoleAuditor, RoleOperator, RoleAdmin:
		return true
	}
	return false
}

// HasPermission checks if the current role has the required permission.
func (r Role) HasPermission(required Role) bool {
	switch required {
	case RoleViewer:
		return r == RoleViewer || r == RoleAuditor || r == RoleOperator || r == RoleAdmin
	case RoleAuditor:
		return r == RoleAuditor || r == RoleOperator || r == RoleAdmin
	case RoleOperator:
		return r == RoleOperator || r == RoleAdmin
	case RoleAdmin:
		return r == RoleAdmin
	}
	return false
}
