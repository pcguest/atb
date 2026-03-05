package main

import (
	"errors"
	"fmt"
	"os"
	"testing"
)

func TestClassifyBundleLoadError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{
			name: "not found maps to user error",
			err:  fmt.Errorf("bundle: load: open: %w", os.ErrNotExist),
			want: exitUserError,
		},
		{
			name: "permission maps to system error",
			err:  fmt.Errorf("bundle: load: open: %w", os.ErrPermission),
			want: exitSystemError,
		},
		{
			name: "unmarshal maps to integrity failure",
			err:  errors.New("bundle: load: unmarshal: invalid character 'x' looking for beginning of value"),
			want: exitIntegrityFailure,
		},
		{
			name: "scan maps to integrity failure",
			err:  errors.New("bundle: load: scan: token too long"),
			want: exitIntegrityFailure,
		},
		{
			name: "other maps to system error",
			err:  errors.New("bundle: load: unknown failure"),
			want: exitSystemError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyBundleLoadError(tc.err)
			if got != tc.want {
				t.Fatalf("unexpected exit code: got %d want %d", got, tc.want)
			}
		})
	}
}
