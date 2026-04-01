package trust

import "testing"

func TestAnchorQualityLabel(t *testing.T) {
	tests := []struct {
		name string
		xc   float64
		ac   float64
		want string
	}{
		{name: "verified", xc: 1.0, ac: 1.0, want: "verified"},
		{name: "digest_only", xc: 0.5, ac: 0.4, want: "digest-only"},
		{name: "present_degraded", xc: 0.1, ac: 0.0, want: "present-degraded"},
		{name: "absent", xc: 0.0, ac: 0.0, want: "absent"},
		{name: "partial", xc: 0.7, ac: 0.3, want: "partial"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := anchorQualityLabel(tc.xc, tc.ac); got != tc.want {
				t.Fatalf("anchorQualityLabel(%.1f, %.1f) = %q, want %q", tc.xc, tc.ac, got, tc.want)
			}
		})
	}
}
