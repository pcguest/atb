// SPDX-License-Identifier: MIT
package main

import (
	"context"
	"path/filepath"
	"slices"
	"testing"

	"github.com/pcguest/atb/internal/event"
	"github.com/pcguest/atb/internal/incident"
	"github.com/pcguest/atb/internal/sessionindex"
	verifypkg "github.com/pcguest/atb/internal/verify"
)

func TestTenonPilotBundleCoversApprovedAndAnomalousActions(t *testing.T) {
	b, err := buildTenonPilotBundle()
	if err != nil {
		t.Fatalf("build Tenon pilot bundle: %v", err)
	}
	if err := b.Verify(); err != nil {
		t.Fatalf("Tenon pilot bundle failed integrity verification: %v", err)
	}

	report, err := verifypkg.EvaluateBundle(verifypkg.EvaluateConfig{
		BundlePath: "tenon-pilot.atb",
		Records:    b.Records,
		Profiles:   []verifypkg.Profile{verifypkg.ProfileByID("atb.profile.privileged_tool_action")},
	})
	if err != nil {
		t.Fatalf("evaluate privileged profile: %v", err)
	}
	if !report.Integrity.ChainValid {
		t.Fatal("integrity: want chain valid")
	}
	if len(report.Profiles) != 1 || !report.Profiles[0].Pass {
		t.Fatalf("privileged profile should pass, got %+v", report.Profiles)
	}
	if report.CAS == nil || report.CAS.Overall <= 0 {
		t.Fatalf("privileged profile should report CAS, got %+v", report.CAS)
	}

	var sawProfileApproval, sawCaptureApproval, sawToolCall, sawActionError bool
	for _, rec := range b.Records {
		switch rec.Event.Type {
		case event.TypeAIHumanApproval:
			sawProfileApproval = true
		case event.TypeHumanApproval:
			sawCaptureApproval = true
		case event.TypeToolCall:
			sawToolCall = true
		case event.TypeAIActionError:
			sawActionError = true
		}
	}
	if !sawProfileApproval || !sawCaptureApproval || !sawToolCall || !sawActionError {
		t.Fatalf("missing expected pilot evidence: ai.human.approval=%v atb.human.approval=%v atb.tool.call=%v ai.action.error=%v",
			sawProfileApproval, sawCaptureApproval, sawToolCall, sawActionError)
	}

	path := filepath.Join(t.TempDir(), "tenon-pilot.atb")
	if err := b.Save(path); err != nil {
		t.Fatalf("save pilot bundle: %v", err)
	}
	entries, err := sessionindex.BuildIndex(context.Background(), []string{path})
	if err != nil {
		t.Fatalf("BuildIndex: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("want 1 session, got %d", len(entries))
	}
	if !slices.Contains(entries[0].AnomalyFlags, "tool_without_approval") {
		t.Fatalf("want tool_without_approval anomaly, got %v", entries[0].AnomalyFlags)
	}

	incidentReport, err := incident.Build(context.Background(), path, pilotSessionID)
	if err != nil {
		t.Fatalf("incident report: %v", err)
	}
	if !incidentReport.Found || !incidentReport.IntegrityValid {
		t.Fatalf("incident report should find a valid session, got found=%v integrity=%v", incidentReport.Found, incidentReport.IntegrityValid)
	}
	if len(incidentReport.Findings) == 0 {
		t.Fatalf("incident report should include findings")
	}
}
