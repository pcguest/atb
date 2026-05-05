// SPDX-License-Identifier: MIT
package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/trust"
	verifypkg "github.com/pcguest/atb/internal/verify"
	apiv1 "github.com/pcguest/atb/pkg/api/v1"
)

func TestInstalledBinarySmokeFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping installed-binary smoke flow in short mode")
	}

	repoRoot := repoRootForInstalledBinarySmoke(t)
	binaryPath := buildInstalledBinary(t, repoRoot)
	bundleDir := t.TempDir()
	bundlePath := filepath.Join(bundleDir, "run.atb", "bundle.atb")
	initResult := runCLIJSON[mutationResult](t, binaryPath, bundleDir, "init", "--format", "json")
	if initResult.Status != "ok" || initResult.Action != "init" {
		t.Fatalf("unexpected init result: %+v", initResult)
	}
	if initResult.Path != bundle.DefaultPath() {
		t.Fatalf("unexpected init path: got %q want %q", initResult.Path, bundle.DefaultPath())
	}

	appendResult := runCLIJSON[mutationResult](
		t,
		binaryPath,
		bundleDir,
		"append",
		"agent.run",
		"--data",
		`{"workflow":"incident-review","case_id":"case-1042"}`,
		"--format",
		"json",
	)
	if appendResult.Status != "ok" || appendResult.Action != "append" {
		t.Fatalf("unexpected append result: %+v", appendResult)
	}
	if appendResult.Sequence != 1 {
		t.Fatalf("unexpected append sequence: got %d want 1", appendResult.Sequence)
	}
	if appendResult.Hash == "" {
		t.Fatalf("expected append hash to be present")
	}

	verifyResult := runCLIJSON[verifypkg.VerifierReport](t, binaryPath, bundleDir, "verify", "--format", "json")
	if verifyResult.BundlePath != bundle.DefaultPath() {
		t.Fatalf("unexpected verify bundle path: got %q want %q", verifyResult.BundlePath, bundle.DefaultPath())
	}
	if verifyResult.ProfileID != "" {
		t.Fatalf("expected no matched profile, got %+v", verifyResult)
	}
	if verifyResult.Pass {
		t.Fatalf("expected no-profile verify report to remain unpassed, got %+v", verifyResult)
	}
	if len(verifyResult.Failures) != 0 {
		t.Fatalf("expected no critical failures for generic workflow, got %+v", verifyResult.Failures)
	}

	trustReport := runCLIJSONExpectExitCode[verifypkg.TrustReport](t, exitUserError, binaryPath, bundleDir, "trust-report", "--format", "json")
	if trustReport.Pass {
		t.Fatalf("expected no-profile trust report to remain unpassed, got %+v", trustReport)
	}
	if trustReport.ProfileID != "" {
		t.Fatalf("expected no matched profile, got %+v", trustReport)
	}
	if trustReport.Chain.RecordCount != 2 {
		t.Fatalf("unexpected trust report chain length: got %d want 2", trustReport.Chain.RecordCount)
	}

	runCLI(
		t,
		binaryPath,
		bundleDir,
		"export",
		"--format",
		"compliance",
		"--output",
		"evidence.zip",
	)

	zr, err := zip.OpenReader(filepath.Join(bundleDir, "evidence.zip"))
	if err != nil {
		t.Fatalf("open export zip: %v", err)
	}
	defer zr.Close()

	names := make([]string, 0, len(zr.File))
	for _, f := range zr.File {
		names = append(names, f.Name)
	}
	requiredEntries := []string{
		"evidence/docs/spec-v1.0.md",
		"evidence/reports/trust-report.json",
		"evidence/reports/verify.json",
	}
	for _, entry := range requiredEntries {
		if !containsString(names, entry) {
			t.Fatalf("missing export entry %q", entry)
		}
	}

	var exportedTrustReport trust.Report
	if err := json.Unmarshal(readZipFile(t, zr.File, "evidence/reports/trust-report.json"), &exportedTrustReport); err != nil {
		t.Fatalf("decode exported trust report: %v", err)
	}
	if exportedTrustReport.Gate.Status != trust.StatusPass {
		t.Fatalf("unexpected exported trust gate status: %+v", exportedTrustReport.Gate)
	}
	if exportedTrustReport.Status != trust.StatusPass {
		t.Fatalf("unexpected exported trust report status: %+v", exportedTrustReport)
	}

	t.Run("view dashboard", func(t *testing.T) {
		port := reserveLocalPort(t)
		viewServer := startInstalledViewServer(t, binaryPath, bundleDir, bundlePath, port)
		defer viewServer.stop()

		viewVerification := waitForViewVerification(t, viewServer, port)
		if viewVerification.Status != "valid" {
			t.Fatalf("unexpected viewer verification status: %+v", viewVerification)
		}
		if viewVerification.ChainLength != 2 {
			t.Fatalf("unexpected viewer chain length: got %d want 2", viewVerification.ChainLength)
		}

		viewResp, viewBody := httpGet(t, fmt.Sprintf("http://127.0.0.1:%d/view/", port))
		if viewResp.StatusCode != http.StatusOK {
			t.Fatalf("unexpected /view/ status: got %d want %d body=%s", viewResp.StatusCode, http.StatusOK, viewBody)
		}
		if viewResp.Header.Get("Content-Security-Policy") == "" {
			t.Fatalf("expected Content-Security-Policy header on /view/")
		}
		if !strings.Contains(viewBody, "Trust Dashboard") {
			t.Fatalf("expected dashboard HTML from installed binary, got %s", viewBody)
		}
	})
}

func repoRootForInstalledBinarySmoke(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("resolve caller path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func buildInstalledBinary(t *testing.T, repoRoot string) string {
	t.Helper()

	binaryPath := filepath.Join(t.TempDir(), "atb")
	if runtime.GOOS == "windows" {
		binaryPath += ".exe"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "build", "-o", binaryPath, "./cmd/atb")
	cmd.Dir = repoRoot
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("build installed binary: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}
	return binaryPath
}

func runCLIJSON[T any](t *testing.T, binaryPath string, workDir string, args ...string) T {
	t.Helper()

	data := runCLI(t, binaryPath, workDir, args...)
	var out T
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("decode JSON for %q: %v\noutput:\n%s", strings.Join(args, " "), err, string(data))
	}
	return out
}

func runCLIJSONExpectExitCode[T any](t *testing.T, wantExitCode int, binaryPath string, workDir string, args ...string) T {
	t.Helper()

	data := runCLIExpectExitCode(t, wantExitCode, binaryPath, workDir, args...)
	var out T
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("decode JSON for %q: %v\noutput:\n%s", strings.Join(args, " "), err, string(data))
	}
	return out
}

func runCLI(t *testing.T, binaryPath string, workDir string, args ...string) []byte {
	t.Helper()

	return runCLIExpectExitCode(t, exitSuccess, binaryPath, workDir, args...)
}

func runCLIExpectExitCode(t *testing.T, wantExitCode int, binaryPath string, workDir string, args ...string) []byte {
	t.Helper()

	return runCLIWithEnvExpectExitCode(t, wantExitCode, binaryPath, workDir, nil, args...)
}

func runCLIWithEnv(t *testing.T, binaryPath string, workDir string, env []string, args ...string) []byte {
	t.Helper()

	return runCLIWithEnvExpectExitCode(t, exitSuccess, binaryPath, workDir, env, args...)
}

func runCLIWithEnvExpectExitCode(t *testing.T, wantExitCode int, binaryPath string, workDir string, env []string, args ...string) []byte {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binaryPath, args...)
	cmd.Dir = workDir
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	exitCode := exitSuccess
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf(
				"run %s %s: %v\nstdout:\n%s\nstderr:\n%s",
				binaryPath,
				strings.Join(args, " "),
				err,
				stdout.String(),
				stderr.String(),
			)
		}
		exitCode = exitErr.ExitCode()
	}
	if exitCode != wantExitCode {
		t.Fatalf(
			"run %s %s: exit code %d, want %d\nstdout:\n%s\nstderr:\n%s",
			binaryPath,
			strings.Join(args, " "),
			exitCode,
			wantExitCode,
			stdout.String(),
			stderr.String(),
		)
	}
	return bytes.TrimSpace(stdout.Bytes())
}

// syncBuffer is a bytes.Buffer wrapped with a mutex for concurrent Write/Read.
// os/exec spawns a copier goroutine that writes to cmd.Stdout while the test
// goroutine reads it; the std bytes.Buffer is not safe for that pattern.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

type runningViewServer struct {
	cmd    *exec.Cmd
	stdout *syncBuffer
	stderr *syncBuffer
	done   chan error
}

func startInstalledViewServer(t *testing.T, binaryPath string, workDir string, bundlePath string, port int) runningViewServer {
	t.Helper()

	cmd := exec.Command(binaryPath, "view", "--bundle", bundlePath, "--no-open", "--port", fmt.Sprintf("%d", port))
	cmd.Dir = workDir
	stdout := &syncBuffer{}
	stderr := &syncBuffer{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("start installed view server: %v", err)
	}

	server := runningViewServer{
		cmd:    cmd,
		stdout: stdout,
		stderr: stderr,
		done:   make(chan error, 1),
	}
	go func() {
		server.done <- cmd.Wait()
	}()
	return server
}

func (s runningViewServer) stop() {
	if s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
	}
	select {
	case <-s.done:
	case <-time.After(5 * time.Second):
	}
}

// extractSessionTokenFromStdout parses the session token printed by the view server
// in the startup URL line: "✓ Serving ... at http://.../#session=<token>"
func extractSessionTokenFromStdout(stdout string) string {
	const marker = "#session="
	idx := strings.Index(stdout, marker)
	if idx < 0 {
		return ""
	}
	rest := stdout[idx+len(marker):]
	end := strings.IndexAny(rest, " \t\r\n")
	if end >= 0 {
		rest = rest[:end]
	}
	return strings.TrimSpace(rest)
}

func waitForViewVerification(t *testing.T, server runningViewServer, port int) apiv1.VerificationResponse {
	t.Helper()

	client := &http.Client{Timeout: 500 * time.Millisecond}
	apiURL := fmt.Sprintf("http://127.0.0.1:%d/api/v1/verification", port)
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case err := <-server.done:
			t.Fatalf(
				"installed view server exited early: %v\nstdout:\n%s\nstderr:\n%s",
				err,
				server.stdout.String(),
				server.stderr.String(),
			)
		default:
		}

		sessionToken := extractSessionTokenFromStdout(server.stdout.String())
		req, err := http.NewRequest(http.MethodGet, apiURL, nil)
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		if sessionToken != "" {
			req.Header.Set("X-ATB-Session-Token", sessionToken)
		}
		resp, err := client.Do(req)
		if err == nil {
			body, readErr := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if readErr == nil && resp.StatusCode == http.StatusOK {
				var out apiv1.VerificationResponse
				if err := json.Unmarshal(body, &out); err == nil {
					return out
				}
			}
		}

		time.Sleep(100 * time.Millisecond)
	}

	t.Fatalf(
		"installed view server did not become ready on port %d\nstdout:\n%s\nstderr:\n%s",
		port,
		server.stdout.String(),
		server.stderr.String(),
	)
	return apiv1.VerificationResponse{}
}

func reserveLocalPort(t *testing.T) int {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		if errors.Is(err, syscall.EPERM) || errors.Is(err, syscall.EACCES) ||
			strings.Contains(strings.ToLower(err.Error()), "operation not permitted") ||
			strings.Contains(strings.ToLower(err.Error()), "permission denied") {
			t.Skipf("cannot bind local smoke-test listener in this environment: %v", err)
		}
		t.Fatalf("reserve local port: %v", err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

func httpGet(t *testing.T, url string) (*http.Response, string) {
	t.Helper()

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		_ = resp.Body.Close()
		t.Fatalf("read %s: %v", url, err)
	}
	if err := resp.Body.Close(); err != nil {
		t.Fatalf("close %s body: %v", url, err)
	}
	return resp, string(body)
}
