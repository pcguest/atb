package auth

import (
	"fmt"
	"testing"
)

func TestRoleIsValid(t *testing.T) {
	tests := []struct {
		role Role
		want bool
	}{
		{RoleViewer, true},
		{RoleAuditor, true},
		{RoleOperator, true},
		{RoleAdmin, true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			if got := tt.role.IsValid(); got != tt.want {
				t.Errorf("Role.IsValid() for %q = %v, want %v", tt.role, got, tt.want)
			}
		})
	}
}

func TestRoleHasPermission(t *testing.T) {
	tests := []struct {
		current  Role
		required Role
		want     bool
	}{
		{RoleViewer, RoleViewer, true},
		{RoleViewer, RoleAuditor, false},
		{RoleViewer, RoleOperator, false},
		{RoleViewer, RoleAdmin, false},

		{RoleAuditor, RoleViewer, true},
		{RoleAuditor, RoleAuditor, true},
		{RoleAuditor, RoleOperator, false},
		{RoleAuditor, RoleAdmin, false},

		{RoleOperator, RoleViewer, true},
		{RoleOperator, RoleAuditor, true},
		{RoleOperator, RoleOperator, true},
		{RoleOperator, RoleAdmin, false},

		{RoleAdmin, RoleViewer, true},
		{RoleAdmin, RoleAuditor, true},
		{RoleAdmin, RoleOperator, true},
		{RoleAdmin, RoleAdmin, true},

		{"invalid", RoleViewer, false},
		{RoleViewer, "invalid", false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_can_do_%s", tt.current, tt.required), func(t *testing.T) {
			if got := tt.current.HasPermission(tt.required); got != tt.want {
				t.Errorf("Role.HasPermission() for current %q, required %q = %v, want %v", tt.current, tt.required, got, tt.want)
			}
		})
	}
}
