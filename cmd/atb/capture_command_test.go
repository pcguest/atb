// SPDX-License-Identifier: MIT
package main

import (
	"bytes"
	"errors"
	"testing"
)

func TestParseCaptureRunArgsBranches(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
		help    bool
	}{
		{name: "help long", args: []string{"--help"}, help: true},
		{name: "missing bundle value", args: []string{"--bundle"}, wantErr: true},
		{name: "duplicate bundle", args: []string{"--bundle", "a", "--bundle", "b"}, wantErr: true},
		{name: "duplicate bundle eq", args: []string{"--bundle=a", "--bundle=b"}, wantErr: true},
		{name: "missing snapshot value", args: []string{"--snapshot"}, wantErr: true},
		{name: "invalid snapshot name", args: []string{"--snapshot", "bad name"}, wantErr: true},
		{name: "missing env prefix value", args: []string{"--env-prefix"}, wantErr: true},
		{name: "invalid env prefix", args: []string{"--env-prefix=1BAD", "--", "x"}, wantErr: true},
		{name: "missing profile value", args: []string{"--profile"}, wantErr: true},
		{name: "invalid lock wait", args: []string{"--lock-wait", "notaduration"}, wantErr: true},
		{name: "invalid lock wait eq", args: []string{"--lock-wait=nope"}, wantErr: true},
		{name: "unknown flag", args: []string{"--nope"}, wantErr: true},
		{name: "missing separator", args: []string{"--bundle", "a"}, wantErr: true},
		{name: "missing command", args: []string{"--"}, wantErr: true},
		{
			name: "valid full",
			args: []string{"--bundle=a.atb", "--snapshot", "snap_1", "--env-prefix", "myapp", "--profile", "p", "--lock-wait", "1s", "--", "run", "arg"},
		},
		{name: "valid eq forms", args: []string{"--snapshot=snap_2", "--env-prefix=MYAPP", "--profile=p2", "--lock-wait=2s", "--", "cmd"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := parseCaptureRunArgs(tc.args)
			if tc.help {
				if !errors.Is(err, errCaptureHelp) {
					t.Fatalf("err = %v, want errCaptureHelp", err)
				}
				return
			}
			if tc.wantErr {
				if err == nil {
					t.Fatalf("want error, got cfg %+v", cfg)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(cfg.Command) == 0 {
				t.Fatal("expected a child command")
			}
		})
	}
}

func TestRunCaptureHandoffArgumentErrors(t *testing.T) {
	cases := [][]string{
		{"--seq"},
		{"--seq", "abc"},
		{"--seq=-1"},
		{"--unknown"},
	}
	for _, args := range cases {
		var out, errBuf bytes.Buffer
		if code := runCaptureHandoff(args, &out, &errBuf); code != exitUserError {
			t.Fatalf("args %v: exit = %d, want %d", args, code, exitUserError)
		}
	}
	var out, errBuf bytes.Buffer
	if code := runCaptureHandoff([]string{"--help"}, &out, &errBuf); code != exitSuccess {
		t.Fatalf("help exit = %d, want success", code)
	}
}

func TestRunCaptureStatusArgumentErrors(t *testing.T) {
	cases := [][]string{
		{"--format"},
		{"--unknown"},
	}
	for _, args := range cases {
		var out, errBuf bytes.Buffer
		if code := runCaptureStatus(args, &out, &errBuf); code != exitUserError {
			t.Fatalf("args %v: exit = %d, want %d", args, code, exitUserError)
		}
	}
	var out, errBuf bytes.Buffer
	if code := runCaptureStatus([]string{"--help"}, &out, &errBuf); code != exitSuccess {
		t.Fatalf("help exit = %d, want success", code)
	}
}

func TestRunCaptureDispatch(t *testing.T) {
	var out, errBuf bytes.Buffer
	if code := runCapture(nil, nil, &out, &errBuf); code != exitUserError {
		t.Fatalf("no subcommand exit = %d, want %d", code, exitUserError)
	}
	out.Reset()
	errBuf.Reset()
	if code := runCapture([]string{"nope"}, nil, &out, &errBuf); code != exitUserError {
		t.Fatalf("unknown subcommand exit = %d, want %d", code, exitUserError)
	}
	out.Reset()
	errBuf.Reset()
	if code := runCapture([]string{"--help"}, nil, &out, &errBuf); code != exitSuccess {
		t.Fatalf("help exit = %d, want success", code)
	}
}
