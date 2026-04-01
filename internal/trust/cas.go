package trust

import (
	"fmt"
	"math"
	"strings"

	"github.com/pcguest/atb/internal/bundle"
	verifypkg "github.com/pcguest/atb/internal/verify"
)

type AnchorQuality struct {
	XC    float64 `json:"xc"`
	AC    float64 `json:"ac"`
	Label string  `json:"label"`
}

type CASSection struct {
	ProfileID     string             `json:"profile_id"`
	WorkflowClass string             `json:"workflow_class"`
	Overall       float64            `json:"overall"`
	Grade         string             `json:"grade"`
	SubScores     map[string]float64 `json:"sub_scores"`
	Weights       map[string]float64 `json:"weight_vector"`
	AnchorQuality AnchorQuality      `json:"anchor_quality"`
}

func anchorQualityLabel(xc, ac float64) string {
	switch {
	case nearlyEqual(xc, 1.0) && nearlyEqual(ac, 1.0):
		return "verified"
	case nearlyEqual(xc, 0.5) && nearlyEqual(ac, 0.4):
		return "digest-only"
	case nearlyEqual(xc, 0.1) && nearlyEqual(ac, 0.0):
		return "present-degraded"
	case nearlyEqual(xc, 0.0) && nearlyEqual(ac, 0.0):
		return "absent"
	default:
		return "partial"
	}
}

func buildCASSection(bundlePath string, profileID string) (*CASSection, string) {
	profileID = strings.TrimSpace(profileID)
	if profileID == "" {
		return nil, ""
	}

	b, err := bundle.Load(bundlePath)
	if err != nil {
		return nil, ""
	}

	profile := verifypkg.ProfileByID(profileID)
	if profile == nil {
		return nil, fmt.Sprintf("Profile %q was not found; CAS section omitted.", profileID)
	}

	profile.Evaluate(b.Records)
	subScores := verifypkg.SubScoresForBundle(b, bundlePath, profile.ID())
	casResult := verifypkg.ComputeCAS(subScores, profile.DefaultWeights(), b.Verify() == nil)

	xc := subScores["XC"]
	ac := subScores["AC"]
	return &CASSection{
		ProfileID:     profile.ID(),
		WorkflowClass: profile.WorkflowClass(),
		Overall:       casResult.Overall,
		Grade:         casResult.Grade,
		SubScores:     subScores,
		Weights:       profile.DefaultWeights(),
		AnchorQuality: AnchorQuality{
			XC:    xc,
			AC:    ac,
			Label: anchorQualityLabel(xc, ac),
		},
	}, ""
}

func casProfileWarningCategory(details string) Category {
	return Category{
		Key:    "completeness_assurance",
		Title:  "Completeness Assurance",
		Status: StatusWarn,
		Checks: []Check{
			{
				ID:       "cas_profile",
				Title:    "CAS Profile Resolution",
				Status:   StatusWarn,
				Severity: SeverityAdvisory,
				Blocking: false,
				Details:  details,
			},
		},
	}
}

func nearlyEqual(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}
