// SPDX-License-Identifier: MIT
// Package insights defines analysis boundaries for Custos evidence review.
package insights

// Pitfall identifies a recurring issue or risk in captured workflow evidence.
type Pitfall struct {
	EventID     string
	Kind        string
	Description string
}

// WorkflowInsight summarises pitfalls and takeaways for a reviewed bundle.
type WorkflowInsight struct {
	Pitfalls  []Pitfall
	Takeaways []string
}

// Analyser extracts workflow-level observations from a bundle.
type Analyser interface {
	// TODO: define analysis inputs, evidence limits, and output confidence semantics.
	Analyse(bundleID string) (*WorkflowInsight, error)
}
