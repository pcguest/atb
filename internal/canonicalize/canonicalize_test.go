package canonicalize

import "testing"

func TestSerializeNumber(t *testing.T) {
	tests := []struct {
		name  string
		value float64
		want  string
	}{
		{
			name:  "uses exponential form at 1e21 boundary",
			value: 1e21,
			want:  "1e+21",
		},
		{
			name:  "uses exponential form below 1e-6",
			value: 1.5e-7,
			want:  "1.5e-07",
		},
		{
			name:  "uses fixed point in normal range",
			value: 123.456,
			want:  "123.456",
		},
		{
			name:  "serializes zero as zero",
			value: 0.0,
			want:  "0",
		},
		{
			name:  "previously wrong large float now uses exponential form",
			value: 1e22,
			want:  "1e+22",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := serializeNumber(tt.value); got != tt.want {
				t.Fatalf("serializeNumber(%v) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}
