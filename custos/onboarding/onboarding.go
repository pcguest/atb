// SPDX-License-Identifier: MIT
// Package onboarding defines organisation provisioning boundaries for Custos.
package onboarding

// OrgProfile captures the organisation and service credentials needed during setup.
type OrgProfile struct {
	OrgID         string
	WorkEmail     string
	PersonalEmail string
	TeamIDs       []string
	APIKeys       []APIKey
}

// APIKey records a service credential reference supplied during onboarding.
type APIKey struct {
	Service string
	Token   string
}

// Onboarder provisions organisation access and connects external services.
type Onboarder interface {
	// TODO: define provisioning side effects and audit requirements.
	Provision(org OrgProfile) error
	// TODO: define service connection and credential storage semantics.
	Connect(orgID, service string) error
}
