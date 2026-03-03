// ATB CLI — Agent Trace Bundle command-line interface.
// Provides commands to initialise, append events to, and verify ATB bundles.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pcguest/atb/internal/bundle"
)

const version = "v0.1.0-dev"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
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
	case "view":
		cmdView()
	case "version", "--version", "-v":
		fmt.Printf("atb %s\n", version)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "atb: unknown command %q\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Print(`ATB — Agent Trace Bundle

Usage:
  atb <command> [flags]

Commands:
  init              Initialise a new ATB bundle in ./run.atb/
  append <type> <json|--data <json>>  Append an event to the current bundle
  snapshot <name> --gate <pass|fail>  Append a snapshot event
  verify [bundle_path]  Verify integrity of a bundle (default: ./run.atb/bundle.atb)
  view [bundle_path] [--port 8080]  Open a local HTML timeline viewer
  version           Print the ATB version

Examples:
  atb init
  atb append dev.session '{"features_built":["hash chaining"]}'
  atb append feature --data '{"name":"atb view"}'
  atb snapshot build --gate pass
  atb verify
  atb view
`)
}

// cmdInit initialises a new bundle directory and empty bundle file.
func cmdInit() {
	path := bundle.DefaultPath()
	if _, err := os.Stat(path); err == nil {
		fmt.Fprintf(os.Stderr, "atb init: bundle already exists at %s\n", path)
		os.Exit(1)
	}
	b := bundle.New()
	if err := b.Save(path); err != nil {
		fmt.Fprintf(os.Stderr, "atb init: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Initialised ATB bundle at %s\n", path)
}

// cmdAppend appends a new event to the existing bundle.
func cmdAppend() {
	if len(os.Args) < 4 {
		fmt.Fprintln(os.Stderr, "Usage: atb append <type> <json|--data <json>>")
		os.Exit(1)
	}
	eventType := os.Args[2]
	rawJSON, err := parseAppendPayload(os.Args[3:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "atb append: %v\n", err)
		os.Exit(1)
	}

	var data interface{}
	if err := json.Unmarshal([]byte(rawJSON), &data); err != nil {
		fmt.Fprintf(os.Stderr, "atb append: invalid JSON: %v\n", err)
		os.Exit(1)
	}

	last, err := appendToDefaultBundle(eventType, data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "atb append: %v\n", err)
		os.Exit(1)
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

func appendToDefaultBundle(eventType string, data interface{}) (bundle.Record, error) {
	path := bundle.DefaultPath()
	b, err := bundle.Load(path)
	if err != nil {
		b = bundle.New()
	}
	if err := b.Append(eventType, data); err != nil {
		return bundle.Record{}, err
	}
	if err := b.Save(path); err != nil {
		return bundle.Record{}, fmt.Errorf("save: %w", err)
	}
	return b.Records[len(b.Records)-1], nil
}

func cmdSnapshot() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: atb snapshot <name> [--gate <pass|fail>]")
		os.Exit(1)
	}
	name := strings.TrimSpace(os.Args[2])
	if name == "" {
		fmt.Fprintln(os.Stderr, "atb snapshot: snapshot name cannot be empty")
		os.Exit(1)
	}
	gate := "pass"
	if len(os.Args) > 3 {
		if len(os.Args) != 5 || os.Args[3] != "--gate" {
			fmt.Fprintln(os.Stderr, "Usage: atb snapshot <name> [--gate <pass|fail>]")
			os.Exit(1)
		}
		g := strings.ToLower(strings.TrimSpace(os.Args[4]))
		if g != "pass" && g != "fail" {
			fmt.Fprintln(os.Stderr, "atb snapshot: --gate must be pass or fail")
			os.Exit(1)
		}
		gate = g
	}
	eventType := fmt.Sprintf("snapshot.%s", name)
	data := map[string]string{
		"gate":      gate,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	last, err := appendToDefaultBundle(eventType, data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "atb snapshot: %v\n", err)
		os.Exit(1)
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

// cmdVerify verifies the integrity of the current bundle.
func cmdVerify() {
	path := bundle.DefaultPath()
	if len(os.Args) >= 3 {
		path = normalizeBundlePath(os.Args[2])
	}
	b, err := bundle.Load(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "atb verify: %v\n", err)
		os.Exit(1)
	}
	if len(b.Records) == 0 {
		fmt.Println("atb verify: bundle is empty — nothing to verify.")
		return
	}
	if err := b.Verify(); err != nil {
		fmt.Fprintf(os.Stderr, "✗ VERIFICATION FAILED: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Bundle verified: %d events, chain intact.\n", len(b.Records))
}
