// ATB CLI — Agent Trace Bundle command-line interface.
// Provides commands to initialise, append events to, and verify ATB bundles.
package main

import (
	"encoding/json"
	"fmt"
	"os"

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
	case "verify":
		cmdVerify()
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
	fmt.Println(`ATB — Agent Trace Bundle

Usage:
  atb <command> [flags]

Commands:
  init              Initialise a new ATB bundle in ./run.atb/
  append <type> <json>  Append an event to the current bundle
  verify            Verify the integrity of the current bundle
  version           Print the ATB version

Examples:
  atb init
  atb append dev.session '{"features_built":["hash chaining"]}'
  atb verify
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
		fmt.Fprintln(os.Stderr, "Usage: atb append <type> <json>")
		os.Exit(1)
	}
	eventType := os.Args[2]
	rawJSON := os.Args[3]

	var data interface{}
	if err := json.Unmarshal([]byte(rawJSON), &data); err != nil {
		fmt.Fprintf(os.Stderr, "atb append: invalid JSON: %v\n", err)
		os.Exit(1)
	}

	path := bundle.DefaultPath()
	b, err := bundle.Load(path)
	if err != nil {
		// If the file doesn't exist, initialise automatically.
		b = bundle.New()
	}

	if err := b.Append(eventType, data); err != nil {
		fmt.Fprintf(os.Stderr, "atb append: %v\n", err)
		os.Exit(1)
	}

	if err := b.Save(path); err != nil {
		fmt.Fprintf(os.Stderr, "atb append: save: %v\n", err)
		os.Exit(1)
	}

	last := b.Records[len(b.Records)-1]
	fmt.Printf("✓ Appended event #%d [%s] hash=%s\n", last.Event.Sequence, last.Event.Type, last.Hash[:16]+"...")
}

// cmdVerify verifies the integrity of the current bundle.
func cmdVerify() {
	path := bundle.DefaultPath()
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
