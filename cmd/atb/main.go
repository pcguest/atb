// ATB CLI — Agent Trace Bundle command-line interface.
// Provides commands to initialise, append events to, and verify ATB bundles.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/hash"
)

const (
	version          = "v0.1.0-dev"
	verifyFormatText = "text"
	verifyFormatJSON = "json"
	verifyAlgorithm  = "SHA-256||RFC8785"
)

type verifyResult struct {
	Status      string `json:"status"`
	ChainLength int    `json:"chain_length"`
	GenesisHash string `json:"genesis_hash"`
	HeadHash    string `json:"head_hash,omitempty"`
	VerifiedAt  string `json:"verified_at"`
	Algorithm   string `json:"algorithm"`
	Path        string `json:"path,omitempty"`
	Error       string `json:"error,omitempty"`
	Message     string `json:"message,omitempty"`
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(exitUserError)
	}
	cmd := os.Args[1]
	switch cmd {
	case "init":
		cmdInit()
	case "append":
		cmdAppend()
	case "snapshot":
		cmdSnapshot()
	case "verify":
		cmdVerify()
	case "trust-report":
		cmdTrustReport()
	case "view":
		cmdView()
	case "version", "--version", "-v":
		fmt.Printf("atb %s\n", version)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "atb: unknown command %q\n", cmd)
		printUsage()
		os.Exit(exitUserError)
	}
}

func printUsage() {
	fmt.Print(`ATB — Agent Trace Bundle

Usage:
  atb <command> [flags]

Commands:
  init [--dry-run]  Initialise a new ATB bundle in ./run.atb/
  append <type> <json|--data <json>> [--dry-run]  Append an event to the current bundle
  snapshot <name> --gate <pass|fail> [--dry-run]  Append a snapshot event
  verify [bundle_path] [--format text|json]  Verify integrity of a bundle (default: ./run.atb/bundle.atb)
  trust-report [bundle_path] [--format markdown|json]  Build a trust report for AI + human audit
  view [bundle_path] [--port 8080]  Open a local HTML timeline viewer
  version           Print the ATB version

Exit codes:
  0  success
  1  user/input error
  2  integrity verification failure
  3  system/runtime error

Examples:
  atb init
  atb init --dry-run
  atb append dev.session '{"features_built":["hash chaining"]}'
  atb append feature --data '{"name":"atb view"}'
  atb append dev.session '{"ok":true}' --dry-run
  atb snapshot build --gate pass
  atb snapshot build --gate pass --dry-run
  atb verify
  atb verify --format json
  atb trust-report --format markdown
  atb trust-report --format json
  atb view
`)
}

// cmdInit initialises a new bundle directory and empty bundle file.
func cmdInit() {
	args, dryRun, err := parseDryRunFlag(os.Args[2:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "atb init: %v\n", err)
		os.Exit(exitUserError)
	}
	if len(args) > 0 {
		fmt.Fprintln(os.Stderr, "Usage: atb init [--dry-run]")
		os.Exit(exitUserError)
	}
	path := bundle.DefaultPath()
	if _, err := os.Stat(path); err == nil {
		if dryRun {
			fmt.Printf("~ Dry run: bundle already exists at %s (no changes).\n", path)
			return
		}
		fmt.Fprintf(os.Stderr, "atb init: bundle already exists at %s\n", path)
		os.Exit(exitUserError)
	}
	if dryRun {
		fmt.Printf("~ Dry run: would initialise ATB bundle at %s.\n", path)
		return
	}
	b := bundle.New()
	if err := b.Save(path); err != nil {
		fmt.Fprintf(os.Stderr, "atb init: %v\n", err)
		os.Exit(exitSystemError)
	}
	fmt.Printf("✓ Initialised ATB bundle at %s\n", path)
}

// cmdAppend appends a new event to the existing bundle.
func cmdAppend() {
	args, dryRun, err := parseDryRunFlag(os.Args[2:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "atb append: %v\n", err)
		os.Exit(exitUserError)
	}
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: atb append <type> <json|--data <json>> [--dry-run]")
		os.Exit(exitUserError)
	}
	eventType := args[0]
	rawJSON, err := parseAppendPayload(args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "atb append: %v\n", err)
		os.Exit(exitUserError)
	}

	var data interface{}
	if err := json.Unmarshal([]byte(rawJSON), &data); err != nil {
		fmt.Fprintf(os.Stderr, "atb append: invalid JSON: %v\n", err)
		os.Exit(exitUserError)
	}

	last, err := appendToDefaultBundle(eventType, data, dryRun)
	if err != nil {
		exitCode := exitSystemError
		var loadErr mutationLoadError
		if errors.As(err, &loadErr) {
			exitCode = classifyBundleLoadError(err)
		}
		fmt.Fprintf(os.Stderr, "atb append: %v\n", err)
		os.Exit(exitCode)
	}
	if dryRun {
		fmt.Printf("~ Dry run: would append event #%d [%s] hash=%s\n", last.Event.Sequence, last.Event.Type, last.Hash[:16]+"...")
		return
	}
	fmt.Printf("✓ Appended event #%d [%s] hash=%s\n", last.Event.Sequence, last.Event.Type, last.Hash[:16]+"...")
}

func parseAppendPayload(args []string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("missing event JSON payload")
	}
	if len(args) == 1 {
		if args[0] == "--data" {
			return "", fmt.Errorf("missing JSON after --data")
		}
		return args[0], nil
	}
	if len(args) == 2 && args[0] == "--data" {
		return args[1], nil
	}
	return "", fmt.Errorf("expected <json> or --data <json>")
}

type mutationLoadError struct {
	err error
}

func (e mutationLoadError) Error() string {
	return fmt.Sprintf("load bundle: %v", e.err)
}

func (e mutationLoadError) Unwrap() error {
	return e.err
}

func parseDryRunFlag(args []string) ([]string, bool, error) {
	filtered := make([]string, 0, len(args))
	dryRun := false
	for _, arg := range args {
		if arg == "--dry-run" {
			dryRun = true
			continue
		}
		filtered = append(filtered, arg)
	}
	return filtered, dryRun, nil
}

func appendToDefaultBundle(eventType string, data interface{}, dryRun bool) (bundle.Record, error) {
	path := bundle.DefaultPath()
	b, err := bundle.Load(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			b = bundle.New()
		} else {
			return bundle.Record{}, mutationLoadError{err: err}
		}
	}
	if err := b.Append(eventType, data); err != nil {
		return bundle.Record{}, err
	}
	last := b.Records[len(b.Records)-1]
	if dryRun {
		return last, nil
	}
	if err := b.Save(path); err != nil {
		return bundle.Record{}, fmt.Errorf("save: %w", err)
	}
	return last, nil
}

func cmdSnapshot() {
	args, dryRun, err := parseDryRunFlag(os.Args[2:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "atb snapshot: %v\n", err)
		os.Exit(exitUserError)
	}
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: atb snapshot <name> [--gate <pass|fail>] [--dry-run]")
		os.Exit(exitUserError)
	}
	name := strings.TrimSpace(args[0])
	if name == "" {
		fmt.Fprintln(os.Stderr, "atb snapshot: snapshot name cannot be empty")
		os.Exit(exitUserError)
	}
	gate := "pass"
	if len(args) > 1 {
		if len(args) != 3 || args[1] != "--gate" {
			fmt.Fprintln(os.Stderr, "Usage: atb snapshot <name> [--gate <pass|fail>] [--dry-run]")
			os.Exit(exitUserError)
		}
		g := strings.ToLower(strings.TrimSpace(args[2]))
		if g != "pass" && g != "fail" {
			fmt.Fprintln(os.Stderr, "atb snapshot: --gate must be pass or fail")
			os.Exit(exitUserError)
		}
		gate = g
	}
	eventType := fmt.Sprintf("snapshot.%s", name)
	data := map[string]string{
		"gate":      gate,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	last, err := appendToDefaultBundle(eventType, data, dryRun)
	if err != nil {
		exitCode := exitSystemError
		var loadErr mutationLoadError
		if errors.As(err, &loadErr) {
			exitCode = classifyBundleLoadError(err)
		}
		fmt.Fprintf(os.Stderr, "atb snapshot: %v\n", err)
		os.Exit(exitCode)
	}
	if dryRun {
		fmt.Printf("~ Dry run: would append snapshot #%d [%s] gate=%s hash=%s\n", last.Event.Sequence, last.Event.Type, gate, last.Hash[:16]+"...")
		return
	}
	fmt.Printf("✓ Appended snapshot #%d [%s] gate=%s hash=%s\n", last.Event.Sequence, last.Event.Type, gate, last.Hash[:16]+"...")
}

func normalizeBundlePath(raw string) string {
	if raw == "" {
		return bundle.DefaultPath()
	}
	if info, err := os.Stat(raw); err == nil && info.IsDir() {
		return filepath.Join(raw, bundle.BundleFile)
	}
	return raw
}

func parseVerifyArgs(args []string) (string, string, error) {
	path := ""
	format := verifyFormatText
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--format":
			if i+1 >= len(args) {
				return "", "", fmt.Errorf("missing value for --format (expected text|json)")
			}
			format = strings.ToLower(strings.TrimSpace(args[i+1]))
			i++
		case strings.HasPrefix(arg, "--format="):
			format = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(arg, "--format=")))
		case strings.HasPrefix(arg, "--"):
			return "", "", fmt.Errorf("unknown flag %q", arg)
		default:
			if path != "" {
				return "", "", fmt.Errorf("verify accepts at most one bundle path")
			}
			path = normalizeBundlePath(arg)
		}
	}
	if path == "" {
		path = bundle.DefaultPath()
	}
	if format != verifyFormatText && format != verifyFormatJSON {
		return "", "", fmt.Errorf("invalid format %q (expected text|json)", format)
	}
	return path, format, nil
}

func newVerifyResult(path string, b *bundle.Bundle, status string) verifyResult {
	result := verifyResult{
		Status:      status,
		ChainLength: 0,
		GenesisHash: hash.GenesisHash,
		VerifiedAt:  time.Now().UTC().Format(time.RFC3339),
		Algorithm:   verifyAlgorithm,
		Path:        path,
	}
	if b == nil {
		return result
	}
	result.ChainLength = len(b.Records)
	if len(b.Records) > 0 {
		result.HeadHash = b.Records[len(b.Records)-1].Hash
	}
	return result
}

func printVerifyJSON(result verifyResult) {
	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		fmt.Fprintf(os.Stderr, "atb verify: encode json output: %v\n", err)
		os.Exit(exitSystemError)
	}
}

// cmdVerify verifies the integrity of the current bundle.
func cmdVerify() {
	path, outputFormat, err := parseVerifyArgs(os.Args[2:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "atb verify: %v\n", err)
		os.Exit(exitUserError)
	}
	b, err := bundle.Load(path)
	if err != nil {
		exitCode := classifyBundleLoadError(err)
		if outputFormat == verifyFormatJSON {
			result := newVerifyResult(path, nil, "error")
			result.Error = err.Error()
			printVerifyJSON(result)
		}
		fmt.Fprintf(os.Stderr, "atb verify: %v\n", err)
		os.Exit(exitCode)
	}
	if len(b.Records) == 0 {
		if outputFormat == verifyFormatJSON {
			result := newVerifyResult(path, b, "empty")
			result.Message = "bundle is empty — nothing to verify"
			printVerifyJSON(result)
			return
		}
		fmt.Println("atb verify: bundle is empty — nothing to verify.")
		return
	}
	if err := b.Verify(); err != nil {
		if outputFormat == verifyFormatJSON {
			result := newVerifyResult(path, b, "invalid")
			result.Error = err.Error()
			printVerifyJSON(result)
		}
		fmt.Fprintf(os.Stderr, "✗ VERIFICATION FAILED: %v\n", err)
		os.Exit(exitIntegrityFailure)
	}
	if outputFormat == verifyFormatJSON {
		printVerifyJSON(newVerifyResult(path, b, "valid"))
		return
	}
	fmt.Printf("✓ Bundle verified: %d events, chain intact.\n", len(b.Records))
}
