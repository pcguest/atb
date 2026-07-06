// SPDX-License-Identifier: MIT
package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunIncidentListAndExportArgumentContracts(t *testing.T) {
	bundlePath := writeIncidentBundle(t)
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "list bundle value", args: []string{"list", "--bundle"}, want: "missing value for --bundle"},
		{name: "list format value", args: []string{"list", "--format"}, want: "missing value for --format"},
		{name: "list unknown", args: []string{"list", "--wat"}, want: "unknown argument"},
		{name: "list format", args: []string{"list", "--bundle", bundlePath, "--format=xml"}, want: "invalid --format"},
		{name: "export bundle value", args: []string{"export", "--bundle"}, want: "missing value for --bundle"},
		{name: "export session value", args: []string{"export", "--session"}, want: "missing value for --session"},
		{name: "export out value", args: []string{"export", "--out"}, want: "missing value for --out"},
		{name: "export endpoint value", args: []string{"export", "--mortise-endpoint"}, want: "missing value for --mortise-endpoint"},
		{name: "export token value", args: []string{"export", "--custos-auth-token"}, want: "missing value for --custos-auth-token"},
		{name: "export unknown", args: []string{"export", "--wat"}, want: "unknown argument"},
		{name: "export bundle required", args: []string{"export", "--session=sess-A", "--out=pack.zip"}, want: "--bundle is required"},
		{name: "export session required", args: []string{"export", "--bundle=" + bundlePath, "--out=pack.zip"}, want: "--session is required"},
		{
			name: "export destination conflict",
			args: []string{"export", "--bundle=" + bundlePath, "--session=sess-A", "--out=pack.zip", "--mortise-endpoint=https://example.test"},
			want: "cannot use both",
		},
		{
			name: "export endpoint aliases conflict",
			args: []string{"export", "--bundle=" + bundlePath, "--session=sess-A", "--mortise-endpoint=https://one.test", "--custos-endpoint=https://two.test"},
			want: "cannot combine",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := runIncident(tc.args, &stdout, &stderr); code != exitUserError {
				t.Fatalf("exit=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
			if !strings.Contains(stderr.String(), tc.want) {
				t.Fatalf("stderr=%q want=%q", stderr.String(), tc.want)
			}
		})
	}

	var stdout, stderr bytes.Buffer
	if code := runIncident([]string{"list", "--help"}, &stdout, &stderr); code != exitSuccess {
		t.Fatalf("list help exit=%d", code)
	}
	stdout.Reset()
	if code := runIncident([]string{"export", "--help"}, &stdout, &stderr); code != exitSuccess {
		t.Fatalf("export help exit=%d", code)
	}
}

func TestRunIncidentExportToCustodyEndpoint(t *testing.T) {
	bundlePath := writeIncidentBundle(t)
	t.Setenv("ATB_MORTISE_TOKEN", "secret")
	var authorization string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"receipt_version":"custos.receipt.v1","receipt_id":"receipt-1","bundle_hash":"sha256:bundle"}`))
	}))
	t.Cleanup(server.Close)

	var stdout, stderr bytes.Buffer
	code := runIncident([]string{
		"export",
		"--bundle=" + bundlePath,
		"--session=sess-A",
		"--mortise-endpoint=" + server.URL,
	}, &stdout, &stderr)
	if code != exitSuccess {
		t.Fatalf("custody exit=%d stderr=%q", code, stderr.String())
	}
	if authorization != "Bearer secret" || !strings.Contains(stdout.String(), "receipt-1") {
		t.Fatalf("authorization=%q stdout=%q", authorization, stdout.String())
	}

	failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	t.Cleanup(failServer.Close)
	stdout.Reset()
	stderr.Reset()
	code = runIncident([]string{
		"export",
		"--bundle=" + bundlePath,
		"--session=sess-A",
		"--mortise-endpoint=" + failServer.URL,
	}, &stdout, &stderr)
	if code != exitSystemError || !strings.Contains(stderr.String(), "push to Mortise") {
		t.Fatalf("failure exit=%d stderr=%q", code, stderr.String())
	}

	t.Setenv("ATB_MORTISE_TOKEN", "")
	t.Setenv("ATB_CUSTOS_TOKEN", "legacy-secret")
	stdout.Reset()
	stderr.Reset()
	code = runIncident([]string{
		"export",
		"--bundle=" + bundlePath,
		"--session=sess-A",
		"--custos-endpoint=" + server.URL,
	}, &stdout, &stderr)
	if code != exitSuccess || authorization != "Bearer legacy-secret" {
		t.Fatalf("legacy alias exit=%d authorization=%q stderr=%q", code, authorization, stderr.String())
	}
}

func TestWriteIncidentPackCreateFailure(t *testing.T) {
	err := writeIncidentPack(filepath.Join(t.TempDir(), "missing", "pack.zip"), nil)
	if err == nil || !strings.Contains(err.Error(), "create") {
		t.Fatalf("create error=%v", err)
	}
}
