// SPDX-License-Identifier: MIT
package evidencepack

import (
	"fmt"
	"io"
	"strings"
	"time"
)

// RenderMarkdown writes a human-readable evidence pack report to w.
func RenderMarkdown(pack Pack, w io.Writer, now time.Time, atbVersion string) error {
	if _, err := fmt.Fprintf(w, "# ATB evidence pack\n\nGenerated at: %s\nATB version: %s\n\n",
		now.UTC().Format(time.RFC3339),
		formatATBVersion(atbVersion),
	); err != nil {
		return err
	}

	for _, bundle := range pack.Bundles {
		if err := renderBundleMarkdown(w, bundle); err != nil {
			return err
		}
	}

	return renderRegulatoryLens(w)
}

func formatATBVersion(version string) string {
	version = strings.TrimSpace(version)
	if version == "" {
		return version
	}
	if strings.HasPrefix(version, "v") {
		return version
	}
	return "v" + version
}

func renderBundleMarkdown(w io.Writer, bundle BundleEvidenceSummary) error {
	if _, err := fmt.Fprintf(w, "## Bundle: %s\n\n", bundle.BundlePath); err != nil {
		return err
	}

	if bundle.Error != "" {
		if _, err := fmt.Fprintf(w, "**Error:** %s\n\n", bundle.Error); err != nil {
			return err
		}
		return nil
	}

	if bundle.ProfileID != "" {
		if _, err := fmt.Fprintf(w, "- Profile: `%s` (v%d)\n", bundle.ProfileID, bundle.ProfileVersion); err != nil {
			return err
		}
	}
	if bundle.HeadHash != "" {
		if _, err := fmt.Fprintf(w, "- Head hash: `%s`\n", bundle.HeadHash); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(w, "- Integrity: %s\n", passFail(bundle.IntegrityPass)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "- Profile obligations: %s\n", passFail(bundle.ProfilePass)); err != nil {
		return err
	}
	if bundle.CASGrade != "" || bundle.CASScore > 0 {
		if _, err := fmt.Fprintf(w, "- CAS: %.2f (%s)\n\n", bundle.CASScore, bundle.CASGrade); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}

	if err := renderResidualRisk(w, bundle); err != nil {
		return err
	}
	return renderExclusions(w, bundle)
}

func renderResidualRisk(w io.Writer, bundle BundleEvidenceSummary) error {
	hasRisk := bundle.ResidualRisk != nil &&
		(bundle.ResidualRisk.Level != "" ||
			len(bundle.ResidualRisk.Drivers) > 0 ||
			len(bundle.ResidualRisk.RecommendedNextEvidence) > 0)
	hasTopLevelNext := len(bundle.RecommendedNextEvidence) > 0
	if !hasRisk && !hasTopLevelNext {
		return nil
	}

	if _, err := fmt.Fprintln(w, "### Residual risk"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}

	if bundle.ResidualRisk != nil && bundle.ResidualRisk.Level != "" {
		if _, err := fmt.Fprintf(w, "- Level: %s\n", bundle.ResidualRisk.Level); err != nil {
			return err
		}
	}

	drivers := []string(nil)
	if bundle.ResidualRisk != nil {
		drivers = bundle.ResidualRisk.Drivers
	}
	if len(drivers) > 0 {
		if _, err := fmt.Fprintln(w, "- Drivers:"); err != nil {
			return err
		}
		for _, driver := range drivers {
			if _, err := fmt.Fprintf(w, "  - %s\n", driver); err != nil {
				return err
			}
		}
	}

	nextEvidence := recommendedNextEvidence(bundle)
	if len(nextEvidence) > 0 {
		if _, err := fmt.Fprintln(w, "- Recommended next evidence:"); err != nil {
			return err
		}
		for _, item := range nextEvidence {
			if _, err := fmt.Fprintf(w, "  - %s\n", item); err != nil {
				return err
			}
		}
	}

	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	return nil
}

func recommendedNextEvidence(bundle BundleEvidenceSummary) []string {
	if bundle.ResidualRisk != nil && len(bundle.ResidualRisk.RecommendedNextEvidence) > 0 {
		return bundle.ResidualRisk.RecommendedNextEvidence
	}
	return bundle.RecommendedNextEvidence
}

func renderExclusions(w io.Writer, bundle BundleEvidenceSummary) error {
	if len(bundle.Exclusions) == 0 {
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
		return nil
	}

	if _, err := fmt.Fprintln(w, "### Exclusions"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	for _, exclusion := range bundle.Exclusions {
		if _, err := fmt.Fprintf(w, "- %s\n", exclusion); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	return nil
}

func renderRegulatoryLens(w io.Writer) error {
	_, err := fmt.Fprintln(w, "## Notes for AI governance / AI Act Article 12",
		"",
		"- Integrity PASS + Profile PASS + CAS High generally indicates strong evidential coverage for traceability.",
		"- Profile FAIL or CAS Low indicates missing evidence against the declared profile; additional logging or integration work is required.",
		"- Exclusions and residual risk drivers highlight potential blind spots in the evidence trail.",
		"",
	)
	return err
}

func passFail(ok bool) string {
	if ok {
		return "PASS"
	}
	return "FAIL"
}
