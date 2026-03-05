package golden

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pcguest/atb/internal/canonicalize"
	"github.com/pcguest/atb/internal/hash"
)

func TestSchemaParity_AllSDKs(t *testing.T) {
	repoRoot := mustRepoRoot(t)

	eventWithFields := map[string]any{
		"seq":          1,
		"prev_hash":    hash.GenesisHash,
		"type":         "schema.parity",
		"data":         map[string]any{"x": 1},
		"actor_id":     "paddy",
		"org_id":       "pcguest",
		"workspace_id": "local",
	}
	eventWithoutFields := map[string]any{
		"seq":       1,
		"prev_hash": hash.GenesisHash,
		"type":      "schema.parity",
		"data":      map[string]any{"x": 1},
	}

	buildTypeScriptForSchemaParity(t, repoRoot)

	goWith := canonicalizeEvent(t, eventWithFields)
	pyWith := runPythonSchemaCanonicalize(t, repoRoot, eventWithFields)
	tsWith := runTypeScriptSchemaCanonicalize(t, repoRoot, eventWithFields)

	if !bytes.Equal(goWith, pyWith) {
		t.Fatalf("go/python schema parity mismatch (with fields)\ngo=%s\npy=%s", goWith, pyWith)
	}
	if !bytes.Equal(goWith, tsWith) {
		t.Fatalf("go/typescript schema parity mismatch (with fields)\ngo=%s\nts=%s", goWith, tsWith)
	}

	goWithout := canonicalizeEvent(t, eventWithoutFields)
	pyWithout := runPythonSchemaCanonicalize(t, repoRoot, eventWithoutFields)
	tsWithout := runTypeScriptSchemaCanonicalize(t, repoRoot, eventWithoutFields)

	if !bytes.Equal(goWithout, pyWithout) {
		t.Fatalf("go/python schema parity mismatch (without fields)\ngo=%s\npy=%s", goWithout, pyWithout)
	}
	if !bytes.Equal(goWithout, tsWithout) {
		t.Fatalf("go/typescript schema parity mismatch (without fields)\ngo=%s\nts=%s", goWithout, tsWithout)
	}

	if bytes.Equal(goWith, goWithout) {
		t.Fatalf(
			"schema parity expected different canonical JSON when optional fields are set\nwith=%s\nwithout=%s",
			goWith,
			goWithout,
		)
	}
}

func canonicalizeEvent(t *testing.T, event map[string]any) []byte {
	t.Helper()
	out, err := canonicalize.Marshal(event)
	if err != nil {
		t.Fatalf("go canonicalize event: %v", err)
	}
	return out
}

func buildTypeScriptForSchemaParity(t *testing.T, repoRoot string) {
	t.Helper()
	tsDir := filepath.Join(repoRoot, "sdk", "typescript")
	build := exec.Command("npm", "run", "build")
	build.Dir = tsDir
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("typescript build for schema parity failed: %v\n%s", err, string(out))
	}
}

func runPythonSchemaCanonicalize(t *testing.T, repoRoot string, event map[string]any) []byte {
	t.Helper()
	payload, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal schema parity event for python: %v", err)
	}

	pythonBin := os.Getenv("ATB_PYTHON_BIN")
	if pythonBin == "" {
		venvPython := filepath.Join(repoRoot, "sdk", "python", "venv", "bin", "python")
		if _, err := os.Stat(venvPython); err == nil {
			pythonBin = venvPython
		} else {
			pythonBin = "python3"
		}
	}

	script := strings.Join([]string{
		"import base64, json, os, pathlib, sys",
		"root = pathlib.Path(os.environ['ATB_REPO_ROOT'])",
		"sys.path.insert(0, str(root / 'sdk' / 'python'))",
		"from atb.canonicalize import canonicalize",
		"event = json.loads(base64.b64decode(os.environ['ATB_SCHEMA_EVENT_B64']).decode('utf-8'))",
		"out = canonicalize(event)",
		"if isinstance(out, bytes): out = out.decode('utf-8')",
		"sys.stdout.write(out)",
	}, "\n")

	cmd := exec.Command(pythonBin, "-c", script)
	cmd.Dir = repoRoot
	cmd.Env = append(
		os.Environ(),
		fmt.Sprintf("ATB_REPO_ROOT=%s", repoRoot),
		fmt.Sprintf("ATB_SCHEMA_EVENT_B64=%s", base64.StdEncoding.EncodeToString(payload)),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("python schema parity command failed: %v\n%s", err, string(out))
	}
	return bytes.TrimSpace(out)
}

func runTypeScriptSchemaCanonicalize(t *testing.T, repoRoot string, event map[string]any) []byte {
	t.Helper()
	payload, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal schema parity event for typescript: %v", err)
	}

	script := strings.Join([]string{
		"const path = require('path');",
		"const root = process.env.ATB_REPO_ROOT;",
		"const { canonicalize } = require(path.join(root, 'sdk', 'typescript', 'dist', 'index.js'));",
		"const event = JSON.parse(Buffer.from(process.env.ATB_SCHEMA_EVENT_B64, 'base64').toString('utf8'));",
		"process.stdout.write(canonicalize(event));",
	}, "\n")

	cmd := exec.Command("node", "-e", script)
	cmd.Dir = repoRoot
	cmd.Env = append(
		os.Environ(),
		fmt.Sprintf("ATB_REPO_ROOT=%s", repoRoot),
		fmt.Sprintf("ATB_SCHEMA_EVENT_B64=%s", base64.StdEncoding.EncodeToString(payload)),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("typescript schema parity command failed: %v\n%s", err, string(out))
	}
	return bytes.TrimSpace(out)
}
