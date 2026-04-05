// ATB CLI — Agent Trace Bundle command-line interface.
// Provides commands to initialise, append events to, and verify ATB bundles.
package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
	"github.com/pcguest/atb/internal/hash"
	signpkg "github.com/pcguest/atb/internal/sign"
)

const (
	version          = "0.9.0-beta"
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

type mutationResult struct {
	Status    string `json:"status"`
	Action    string `json:"action"`
	DryRun    bool   `json:"dry_run"`
	Path      string `json:"path"`
	Sequence  int    `json:"sequence,omitempty"`
	EventType string `json:"event_type,omitempty"`
	Gate      string `json:"gate,omitempty"`
	Hash      string `json:"hash,omitempty"`
	Message   string `json:"message,omitempty"`
	Error     string `json:"error,omitempty"`
	ExitCode  int    `json:"exit_code,omitempty"`
}

type helpCommand struct {
	Name        string   `json:"name"`
	Usage       string   `json:"usage"`
	Description string   `json:"description"`
	Flags       []string `json:"flags,omitempty"`
	Mutating    bool     `json:"mutating"`
}

type helpOutput struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Description string            `json:"description"`
	Usage       string            `json:"usage"`
	ExitCodes   map[string]string `json:"exit_codes"`
	Commands    []helpCommand     `json:"commands"`
}

func usageJSON() helpOutput {
	return helpOutput{
		Name:        "atb",
		Version:     version,
		Description: "ATB — Agent Trace Bundle",
		Usage:       "atb <command> [flags]",
		ExitCodes: map[string]string{
			"0": "success",
			"1": "user/input error",
			"2": "integrity verification failure",
			"3": "profile verification failure or system/runtime error",
		},
		Commands: []helpCommand{
			{
				Name:        "init",
				Usage:       "atb init [--dry-run] [--format text|json]",
				Description: "Initialise a new ATB bundle (idempotent).",
				Flags:       []string{"--dry-run", "--format"},
				Mutating:    true,
			},
			{
				Name:        "bundle",
				Usage:       "atb bundle new [--dry-run] [--format text|json]",
				Description: "Initialise a new ATB bundle (alias for init).",
				Flags:       []string{"--dry-run", "--format"},
				Mutating:    true,
			},
			{
				Name:        "append",
				Usage:       "atb append <type> <json|--data <json>> [--actor-id <id>] [--org-id <id>] [--workspace-id <id>] [--sign-policy <path>] [--dry-run] [--format text|json]",
				Description: "Append an event to the current bundle.",
				Flags:       []string{"--data", "--actor-id", "--org-id", "--workspace-id", "--sign-policy", "--dry-run", "--format"},
				Mutating:    true,
			},
			{
				Name:        "snapshot",
				Usage:       "atb snapshot <name> [--dry-run] [--format text|json]",
				Description: "Append a snapshot event to the current bundle.",
				Flags:       []string{"--dry-run", "--format"},
				Mutating:    true,
			},
			{
				Name:        "anchor",
				Usage:       "atb anchor [bundle_path] [--tsa-url <url>]",
				Description: "Submit the current bundle hash to an RFC 3161 TSA and save the token.",
				Flags:       []string{"--tsa-url"},
				Mutating:    true,
			},
			{
				Name:        "keygen",
				Usage:       "atb keygen [--out-dir <dir>]",
				Description: "Generate an Ed25519 signing keypair.",
				Flags:       []string{"--out-dir"},
				Mutating:    true,
			},
			{
				Name:        "sign",
				Usage:       "atb sign --bundle <path> --key <path> [--out <path>]",
				Description: "Append an Ed25519 bundle signature record.",
				Flags:       []string{"--bundle", "--key", "--out"},
				Mutating:    true,
			},
			{
				Name:        "verify",
				Usage:       "atb verify [bundle_path] [--bundle <path>] [--profile <id|path>] [--json] [--format text|json] [--quiet] [--trace] [--with-anchor] [--roots <pem-file>]",
				Description: "Verify bundle integrity and evaluate obligation profiles.",
				Flags:       []string{"--bundle", "--profile", "--json", "--format", "--quiet", "--trace", "--with-anchor", "--roots"},
				Mutating:    false,
			},
			{
				Name:        "inspect",
				Usage:       "atb inspect [bundle_path] [--bundle <path>] [--json] [--seq <n>]",
				Description: "Inspect bundle records in table or JSON form.",
				Flags:       []string{"--bundle", "--json", "--seq"},
				Mutating:    false,
			},
			{
				Name:        "events",
				Usage:       "atb events [--json] [--profile <id>]",
				Description: "List canonical ATB event types.",
				Flags:       []string{"--json", "--profile"},
				Mutating:    false,
			},
			{
				Name:        "encrypt",
				Usage:       "atb encrypt [bundle_path] [--output <path>] [--password <password>]",
				Description: "Encrypt a bundle using AES-256-GCM.",
				Flags:       []string{"--output", "--password"},
				Mutating:    true,
			},
			{
				Name:        "decrypt",
				Usage:       "atb decrypt <encrypted_path> [--output <path>] [--password <password>]",
				Description: "Decrypt an encrypted ATB bundle.",
				Flags:       []string{"--output", "--password"},
				Mutating:    true,
			},
			{
				Name:        "archive",
				Usage:       "atb archive [--before YYYY-MM-DD] [--dry-run]",
				Description: "Archive old bundles and append tamper-evident ledger entries.",
				Flags:       []string{"--before", "--dry-run"},
				Mutating:    true,
			},
			{
				Name:        "export",
				Usage:       "atb export --format <compliance|soc2|gdpr> --output <path.zip> [--bundle <path>] [--type dsr|ropa] [--subject-id <id>] [--dry-run] [--json] [--with-verify]",
				Description: "Export local compliance evidence bundle.",
				Flags:       []string{"--format", "--output", "--bundle", "--type", "--subject-id", "--dry-run", "--json", "--with-verify"},
				Mutating:    false,
			},
			{
				Name:        "config",
				Usage:       "atb config retention --days <n>",
				Description: "Set local ATB configuration values.",
				Flags:       []string{"--days"},
				Mutating:    true,
			},
			{
				Name:        "trust-report",
				Usage:       "atb trust-report [bundle_path] [--format markdown|json|text] [--profile <id>]",
				Description: "Generate trust report sections for audit.",
				Flags:       []string{"--format", "--profile"},
				Mutating:    false,
			},
			{
				Name:        "view",
				Usage:       "atb view [bundle_path] [--bundle path/to/file.atb] [--port 8080] [--no-open] [--log-reveals] [--ui-experimental]",
				Description: "Open the local viewer. Add --ui-experimental for the dashboard preview.",
				Flags:       []string{"--bundle", "--port", "--no-open", "--log-reveals", "--ui-experimental"},
				Mutating:    false,
			},
			{
				Name:        "doc",
				Usage:       "atb doc gen-openapi [--output docs/api/openapi.yaml]",
				Description: "Generate API documentation artifacts.",
				Flags:       []string{"--output"},
				Mutating:    false,
			},
			{
				Name:        "version",
				Usage:       "atb version",
				Description: "Print ATB version.",
				Mutating:    false,
			},
		},
	}
}

func printUsageJSON() {
	if err := json.NewEncoder(os.Stdout).Encode(usageJSON()); err != nil {
		fmt.Fprintf(os.Stderr, "atb help: encode json output: %v\n", err)
		os.Exit(exitSystemError)
	}
}

func parseHelpArgs(args []string) (string, error) {
	format := "text"
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--format":
			if i+1 >= len(args) {
				return "", fmt.Errorf("missing value for --format (expected text|json)")
			}
			format = strings.ToLower(strings.TrimSpace(args[i+1]))
			i++
		case strings.HasPrefix(arg, "--format="):
			format = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(arg, "--format=")))
		case strings.HasPrefix(arg, "-"):
			return "", fmt.Errorf("unknown flag %q", arg)
		default:
			return "", fmt.Errorf("unexpected argument %q", arg)
		}
	}
	if format != "text" && format != "json" {
		return "", fmt.Errorf("invalid format %q (expected text|json)", format)
	}
	return format, nil
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
	case "bundle":
		cmdBundle()
	case "append":
		cmdAppend()
	case "snapshot":
		cmdSnapshot()
	case "anchor":
		cmdAnchor()
	case "keygen":
		cmdKeygen()
	case "sign":
		cmdSign()
	case "verify":
		cmdVerifyProfile()
	case "inspect":
		cmdInspect()
	case "events":
		cmdEvents()
	case "encrypt":
		cmdEncrypt()
	case "decrypt":
		cmdDecrypt()
	case "archive":
		cmdArchive()
	case "export":
		cmdExport()
	case "config":
		cmdConfig()
	case "trust-report":
		cmdTrustReport()
	case "view":
		cmdView()
	case "doc":
		cmdDoc()
	case "version", "--version", "-v":
		fmt.Printf("atb %s\n", version)
	case "help", "--help", "-h":
		format, err := parseHelpArgs(os.Args[2:])
		if err != nil {
			fmt.Fprintf(os.Stderr, "atb help: %v\n", err)
			os.Exit(exitUserError)
		}
		if format == "json" {
			printUsageJSON()
			return
		}
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
  init [--dry-run] [--format text|json]  Initialise a new ATB bundle in ./run.atb/ (idempotent)
  bundle new [--dry-run] [--format text|json]  Initialise a new ATB bundle in ./run.atb/ (alias for init)
  append <type> <json|--data <json>> [--actor-id <id>] [--org-id <id>] [--workspace-id <id>] [--sign-policy <path>] [--dry-run] [--format text|json]  Append an event to the current bundle
  snapshot <name> [--dry-run] [--format text|json]  Append a snapshot event
  anchor [bundle_path] [--tsa-url <url>]  Submit the current bundle hash to an RFC 3161 TSA and save the token
  keygen [--out-dir <dir>]  Generate an Ed25519 signing keypair
  sign --bundle <path> --key <path> [--out <path>]  Append an Ed25519 bundle signature record
  verify [bundle_path] [--bundle <path>] [--profile <id|path>] [--json] [--format text|json] [--quiet] [--trace] [--with-anchor] [--roots <pem-file>]  Verify integrity of a bundle and evaluate obligation profiles
  inspect [bundle_path] [--bundle <path>] [--json] [--seq <n>]  Inspect bundle records in table or JSON form
  events [--json] [--profile <id>]  List canonical ATB event types
  encrypt [bundle_path] [--output <path>] [--password <password>]  Encrypt bundle file to <bundle_path>.enc or a chosen path
  decrypt <encrypted_path> [--output <path>] [--password <password>]  Decrypt encrypted bundle to the default or chosen path
  archive [--before YYYY-MM-DD] [--dry-run]  Archive old bundles into ./archive.atb/ with ledger entries
  export --format <compliance|soc2|gdpr> --output <path.zip> [--bundle <path>] [--type dsr|ropa] [--subject-id <id>] [--dry-run] [--json] [--with-verify]  Export auditor-friendly local evidence bundle
  config retention --days <n>  Set local retention policy config in ./.atb/config.json
  trust-report [bundle_path] [--format markdown|json|text] [--profile <id>]  Build a trust report for AI + human audit
  view [bundle_path] [--bundle path/to/file.atb] [--port 8080] [--no-open] [--log-reveals] [--ui-experimental]  Open the local viewer (dashboard preview behind --ui-experimental)
  doc gen-openapi [--output docs/api/openapi.yaml]  Generate API docs artifacts
  version           Print the ATB version

Exit codes:
  0  success
  1  user/input error
  2  integrity verification failure
  3  profile verification failure or system/runtime error

Examples:
  atb init
  atb init --dry-run
  atb init --format json
  atb bundle new
  atb bundle new --dry-run
  atb bundle new --format json
  atb append dev.session '{"features_built":["hash chaining"]}'
  atb append feature --data '{"name":"atb view"}'
  atb append dev.session --data '{"x":1}' --actor-id paddy --org-id pcguest --workspace-id local
  atb append ai.policy.decision --data '{"policy_id":"pol-1","policy_version":"2026-04","decision":"allow","decision_reason_codes":["ticket_present"],"subject_id_hash":"subject-hash","action_id":"act-1"}' --sign-policy ./atb-key.pem
  atb append dev.session '{"ok":true}' --dry-run
  atb append dev.session '{"ok":true}' --format json
  atb snapshot build_complete
  atb snapshot build_complete --dry-run
  atb snapshot build_complete --format json
  atb anchor
  atb anchor --tsa-url http://timestamp.digicert.com
  atb keygen --out-dir ./keys
  atb sign --bundle run.atb/bundle.atb --key ./atb-key.pem
  atb verify
  atb verify --json
  atb verify --quiet
  atb verify --bundle run.atb/bundle.atb --profile atb.profile.privileged_tool_action
  atb verify --bundle run.atb/bundle.atb --profile ./my-profile.yaml
  atb verify --format json
  atb verify --trace
  atb verify --with-anchor
  atb verify --with-anchor --roots ./tsa-roots.pem
  atb inspect --bundle run.atb/bundle.atb
  atb inspect --bundle run.atb/bundle.atb --seq 0
  atb inspect --bundle run.atb/bundle.atb --json
  atb events
  atb events --json
  atb events --profile atb.profile.rag_answer
  ATB_PASSWORD=test123 atb encrypt run.atb/bundle.atb --output handoff/acme-review.atb.enc
  ATB_PASSWORD=test123 atb decrypt handoff/acme-review.atb.enc --output review/acme-review.atb
  atb archive --before 2025-01-01 --dry-run
  atb export --format compliance --output evidence.zip --dry-run
  atb export --format soc2 --bundle run.atb/bundle.atb --output soc2-evidence.zip
  atb export --format gdpr --type dsr --subject-id usr_123 --bundle run.atb/bundle.atb --output gdpr-dsr.zip
  atb export --format gdpr --type ropa --bundle run.atb/bundle.atb --output gdpr-ropa.zip
  atb config retention --days 90
  atb trust-report --format markdown
  atb trust-report --format json
  atb trust-report --format json --profile atb.profile.privileged_tool_action
  atb view
  atb view --ui-experimental
  atb view --bundle run.atb/bundle.atb --port 8080 --no-open
  atb doc gen-openapi
`)
}

// cmdInit initialises a new bundle directory and empty bundle file.
func cmdInit() {
	os.Exit(runInit(os.Args[2:], os.Stdout, os.Stderr))
}

func runInit(args []string, stdout, stderr io.Writer) int {
	rawArgs := append([]string(nil), args...)
	args, outputFormat, dryRun, err := parseMutationFlags(args)
	if err != nil {
		joinedArgs := strings.Join(rawArgs, " ")
		if strings.Contains(joinedArgs, "--format json") || strings.Contains(joinedArgs, "--format=json") {
			printMutationJSON(mutationResult{
				Status:   "error",
				Action:   "init",
				DryRun:   dryRun,
				Path:     bundle.DefaultPath(),
				Error:    err.Error(),
				ExitCode: exitUserError,
			}, "init")
			return exitUserError
		}
		fmt.Fprintf(stderr, "atb init: %v\n", err)
		return exitUserError
	}
	if len(args) > 0 {
		if outputFormat == verifyFormatJSON {
			printMutationJSON(mutationResult{
				Status:   "error",
				Action:   "init",
				DryRun:   dryRun,
				Path:     bundle.DefaultPath(),
				Error:    "usage: atb init [--dry-run] [--format text|json]",
				ExitCode: exitUserError,
			}, "init")
			return exitUserError
		}
		fmt.Fprintln(stderr, "Usage: atb init [--dry-run] [--format text|json]")
		return exitUserError
	}
	path := bundle.DefaultPath()
	if _, err := os.Stat(path); err == nil {
		if dryRun {
			if outputFormat == verifyFormatJSON {
				printMutationJSON(mutationResult{
					Status:  "ok",
					Action:  "noop",
					DryRun:  true,
					Path:    path,
					Message: "bundle already exists; no changes",
				}, "init")
				return exitSuccess
			}
			fmt.Fprintf(stdout, "~ Dry run: bundle already exists at %s (no changes).\n", path)
			return exitSuccess
		}
		if outputFormat == verifyFormatJSON {
			printMutationJSON(mutationResult{
				Status:  "ok",
				Action:  "noop",
				DryRun:  false,
				Path:    path,
				Message: "bundle already exists; no changes",
			}, "init")
			return exitSuccess
		}
		fmt.Fprintf(stdout, "atb init: bundle already exists at %s (no changes).\n", path)
		return exitSuccess
	} else if !errors.Is(err, os.ErrNotExist) {
		if outputFormat == verifyFormatJSON {
			printMutationJSON(mutationResult{
				Status:   "error",
				Action:   "init",
				DryRun:   dryRun,
				Path:     path,
				Error:    fmt.Sprintf("stat %s: %v", path, err),
				ExitCode: exitSystemError,
			}, "init")
			return exitSystemError
		}
		fmt.Fprintf(stderr, "atb init: stat %s: %v\n", path, err)
		return exitSystemError
	}
	if dryRun {
		if outputFormat == verifyFormatJSON {
			printMutationJSON(mutationResult{
				Status:  "ok",
				Action:  "init",
				DryRun:  true,
				Path:    path,
				Message: "bundle would be initialised",
			}, "init")
			return exitSuccess
		}
		fmt.Fprintf(stdout, "~ Dry run: would initialise ATB bundle at %s.\n", path)
		return exitSuccess
	}
	b, err := bundle.New()
	if err != nil {
		if outputFormat == verifyFormatJSON {
			printMutationJSON(mutationResult{
				Status:   "error",
				Action:   "init",
				DryRun:   dryRun,
				Path:     path,
				Error:    err.Error(),
				ExitCode: exitSystemError,
			}, "init")
			return exitSystemError
		}
		fmt.Fprintf(stderr, "atb init: %v\n", err)
		return exitSystemError
	}
	if err := b.Save(path); err != nil {
		if outputFormat == verifyFormatJSON {
			printMutationJSON(mutationResult{
				Status:   "error",
				Action:   "init",
				DryRun:   dryRun,
				Path:     path,
				Error:    err.Error(),
				ExitCode: exitSystemError,
			}, "init")
			return exitSystemError
		}
		fmt.Fprintf(stderr, "atb init: %v\n", err)
		return exitSystemError
	}
	if outputFormat == verifyFormatJSON {
		printMutationJSON(mutationResult{
			Status:  "ok",
			Action:  "init",
			DryRun:  false,
			Path:    path,
			Message: "bundle initialised",
		}, "init")
		return exitSuccess
	}
	fmt.Fprintf(stdout, "✓ Initialised ATB bundle at %s\n", path)
	return exitSuccess
}

func cmdAppend() {
	os.Exit(runAppend(os.Args[2:], os.Stdout, os.Stderr))
}

// runAppend appends a new event to the existing bundle.
func runAppend(args []string, stdout, stderr io.Writer) int {
	rawArgs := append([]string(nil), args...)
	args, outputFormat, dryRun, err := parseMutationFlags(args)
	if err != nil {
		if strings.Contains(strings.Join(rawArgs, " "), "--format json") || strings.Contains(strings.Join(rawArgs, " "), "--format=json") {
			printMutationJSON(mutationResult{
				Status:   "error",
				Action:   "append",
				DryRun:   dryRun,
				Path:     bundle.DefaultPath(),
				Error:    err.Error(),
				ExitCode: exitUserError,
			}, "append")
			return exitUserError
		}
		fmt.Fprintf(stderr, "atb append: %v\n", err)
		return exitUserError
	}
	if len(args) < 2 {
		if outputFormat == verifyFormatJSON {
			printMutationJSON(mutationResult{
				Status:   "error",
				Action:   "append",
				DryRun:   dryRun,
				Path:     bundle.DefaultPath(),
				Error:    "usage: atb append <type> <json|--data <json>> [--actor-id <id>] [--org-id <id>] [--workspace-id <id>] [--sign-policy <path>] [--dry-run] [--format text|json]",
				ExitCode: exitUserError,
			}, "append")
			return exitUserError
		}
		fmt.Fprintln(stderr, "Usage: atb append <type> <json|--data <json>> [--actor-id <id>] [--org-id <id>] [--workspace-id <id>] [--sign-policy <path>] [--dry-run] [--format text|json]")
		return exitUserError
	}
	eventType := args[0]
	appendInput, err := parseAppendCommandArgs(args[1:])
	if err != nil {
		if outputFormat == verifyFormatJSON {
			printMutationJSON(mutationResult{
				Status:    "error",
				Action:    "append",
				DryRun:    dryRun,
				Path:      bundle.DefaultPath(),
				EventType: eventType,
				Error:     err.Error(),
				ExitCode:  exitUserError,
			}, "append")
			return exitUserError
		}
		fmt.Fprintf(stderr, "atb append: %v\n", err)
		return exitUserError
	}
	rawJSON := appendInput.RawJSON

	var data interface{}
	if err := json.Unmarshal([]byte(rawJSON), &data); err != nil {
		if outputFormat == verifyFormatJSON {
			printMutationJSON(mutationResult{
				Status:    "error",
				Action:    "append",
				DryRun:    dryRun,
				Path:      bundle.DefaultPath(),
				EventType: eventType,
				Error:     fmt.Sprintf("invalid JSON: %v", err),
				ExitCode:  exitUserError,
			}, "append")
			return exitUserError
		}
		fmt.Fprintf(stderr, "atb append: invalid JSON: %v\n", err)
		return exitUserError
	}

	if err := maybeSignPolicyDecisionEvent(eventType, data, appendInput.SignPolicyKeyPath, stderr); err != nil {
		exitCode := classifyAppendPolicySignError(err)
		if outputFormat == verifyFormatJSON {
			printMutationJSON(mutationResult{
				Status:    "error",
				Action:    "append",
				DryRun:    dryRun,
				Path:      bundle.DefaultPath(),
				EventType: eventType,
				Error:     err.Error(),
				ExitCode:  exitCode,
			}, "append")
			return exitCode
		}
		fmt.Fprintf(stderr, "atb append: %v\n", err)
		return exitCode
	}

	last, err := appendToDefaultBundle(eventType, data, dryRun, appendInput.Options)
	if err != nil {
		exitCode := exitSystemError
		var loadErr mutationLoadError
		if errors.As(err, &loadErr) {
			exitCode = classifyBundleLoadError(err)
		}
		if outputFormat == verifyFormatJSON {
			printMutationJSON(mutationResult{
				Status:    "error",
				Action:    "append",
				DryRun:    dryRun,
				Path:      bundle.DefaultPath(),
				EventType: eventType,
				Error:     err.Error(),
				ExitCode:  exitCode,
			}, "append")
			return exitCode
		}
		fmt.Fprintf(stderr, "atb append: %v\n", err)
		return exitCode
	}
	if outputFormat == verifyFormatJSON {
		action := "append"
		message := "event appended"
		if dryRun {
			action = "preview_append"
			message = "event would be appended"
		}
		printMutationJSON(mutationResult{
			Status:    "ok",
			Action:    action,
			DryRun:    dryRun,
			Path:      bundle.DefaultPath(),
			Sequence:  last.Event.Sequence,
			EventType: last.Event.Type,
			Hash:      last.Hash,
			Message:   message,
		}, "append")
		return exitSuccess
	}
	if dryRun {
		fmt.Fprintf(stdout, "~ Dry run: would append event #%d [%s] hash=%s\n", last.Event.Sequence, last.Event.Type, last.Hash[:16]+"...")
		return exitSuccess
	}
	fmt.Fprintf(stdout, "✓ Appended event #%d [%s] hash=%s\n", last.Event.Sequence, last.Event.Type, last.Hash[:16]+"...")
	return exitSuccess
}

func parseAppendPayload(args []string) (string, error) {
	input, err := parseAppendCommandArgs(args)
	if err != nil {
		return "", err
	}
	return input.RawJSON, nil
}

type appendCommandInput struct {
	RawJSON           string
	Options           bundle.AppendOptions
	SignPolicyKeyPath string
}

func parseAppendCommandArgs(args []string) (appendCommandInput, error) {
	result := appendCommandInput{}
	setIDField := func(flagName string, target **string, value string) error {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return fmt.Errorf("%s cannot be empty", flagName)
		}
		v := trimmed
		*target = &v
		return nil
	}

	if len(args) == 0 {
		return result, fmt.Errorf("missing event JSON payload")
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--data":
			if i+1 >= len(args) {
				return result, fmt.Errorf("missing JSON after --data")
			}
			if result.RawJSON != "" {
				return result, fmt.Errorf("expected <json> or --data <json>")
			}
			result.RawJSON = args[i+1]
			i++
		case strings.HasPrefix(arg, "--data="):
			if result.RawJSON != "" {
				return result, fmt.Errorf("expected <json> or --data <json>")
			}
			result.RawJSON = strings.TrimPrefix(arg, "--data=")
			if result.RawJSON == "" {
				return result, fmt.Errorf("missing JSON after --data")
			}
		case arg == "--actor-id":
			if i+1 >= len(args) {
				return result, fmt.Errorf("missing value for --actor-id")
			}
			if err := setIDField("--actor-id", &result.Options.ActorID, args[i+1]); err != nil {
				return result, err
			}
			i++
		case strings.HasPrefix(arg, "--actor-id="):
			if err := setIDField("--actor-id", &result.Options.ActorID, strings.TrimPrefix(arg, "--actor-id=")); err != nil {
				return result, err
			}
		case arg == "--org-id":
			if i+1 >= len(args) {
				return result, fmt.Errorf("missing value for --org-id")
			}
			if err := setIDField("--org-id", &result.Options.OrgID, args[i+1]); err != nil {
				return result, err
			}
			i++
		case strings.HasPrefix(arg, "--org-id="):
			if err := setIDField("--org-id", &result.Options.OrgID, strings.TrimPrefix(arg, "--org-id=")); err != nil {
				return result, err
			}
		case arg == "--workspace-id":
			if i+1 >= len(args) {
				return result, fmt.Errorf("missing value for --workspace-id")
			}
			if err := setIDField("--workspace-id", &result.Options.WorkspaceID, args[i+1]); err != nil {
				return result, err
			}
			i++
		case strings.HasPrefix(arg, "--workspace-id="):
			if err := setIDField("--workspace-id", &result.Options.WorkspaceID, strings.TrimPrefix(arg, "--workspace-id=")); err != nil {
				return result, err
			}
		case arg == "--sign-policy":
			if i+1 >= len(args) {
				return result, fmt.Errorf("missing value for --sign-policy")
			}
			result.SignPolicyKeyPath = filepath.Clean(strings.TrimSpace(args[i+1]))
			if result.SignPolicyKeyPath == "." {
				return result, fmt.Errorf("--sign-policy cannot be empty")
			}
			i++
		case strings.HasPrefix(arg, "--sign-policy="):
			result.SignPolicyKeyPath = filepath.Clean(strings.TrimSpace(strings.TrimPrefix(arg, "--sign-policy=")))
			if result.SignPolicyKeyPath == "." {
				return result, fmt.Errorf("--sign-policy cannot be empty")
			}
		case strings.HasPrefix(arg, "--"):
			return result, fmt.Errorf("unknown flag %q", arg)
		default:
			if result.RawJSON != "" {
				return result, fmt.Errorf("expected <json> or --data <json>")
			}
			result.RawJSON = arg
		}
	}
	if result.RawJSON == "" {
		return result, fmt.Errorf("missing event JSON payload")
	}
	return result, nil
}

func maybeSignPolicyDecisionEvent(eventType string, data interface{}, keyPath string, stderr io.Writer) error {
	if keyPath == "" {
		return nil
	}
	if eventType != event.TypeAIPolicyDecision {
		fmt.Fprintf(stderr, "atb append: warning: --sign-policy ignored for %s\n", eventType)
		return nil
	}

	fields, ok := data.(map[string]any)
	if !ok {
		return fmt.Errorf("ai.policy.decision payload must be a JSON object when --sign-policy is set")
	}

	privateKey, err := loadEd25519PrivateKey(keyPath)
	if err != nil {
		return err
	}

	signature, err := signpkg.SignPolicyDecision(fields, privateKey)
	if err != nil {
		return err
	}

	publicKey, ok := privateKey.Public().(ed25519.PublicKey)
	if !ok {
		return fmt.Errorf("derive public key: loaded key is not Ed25519")
	}

	fields[event.FieldPolicySignature] = signature
	fields[event.FieldPolicySignerPubKey] = base64.StdEncoding.EncodeToString(publicKey)
	return nil
}

func classifyAppendPolicySignError(err error) int {
	if strings.Contains(strings.ToLower(err.Error()), "json object") {
		return exitUserError
	}
	return classifySignError(err)
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

func printMutationJSON(result mutationResult, command string) {
	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		fmt.Fprintf(os.Stderr, "atb %s: encode json output: %v\n", command, err)
		os.Exit(exitSystemError)
	}
}

func parseMutationFlags(args []string) ([]string, string, bool, error) {
	filtered := make([]string, 0, len(args))
	outputFormat := verifyFormatText
	dryRun := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--dry-run":
			dryRun = true
		case arg == "--format":
			if i+1 >= len(args) {
				return nil, "", false, fmt.Errorf("missing value for --format (expected text|json)")
			}
			outputFormat = strings.ToLower(strings.TrimSpace(args[i+1]))
			i++
		case strings.HasPrefix(arg, "--format="):
			outputFormat = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(arg, "--format=")))
		default:
			filtered = append(filtered, arg)
		}
	}
	if outputFormat != verifyFormatText && outputFormat != verifyFormatJSON {
		return nil, "", false, fmt.Errorf("invalid format %q (expected text|json)", outputFormat)
	}
	return filtered, outputFormat, dryRun, nil
}

func appendToDefaultBundle(eventType string, data interface{}, dryRun bool, opts ...bundle.AppendOptions) (bundle.Record, error) {
	path := bundle.DefaultPath()
	b, err := bundle.Load(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			b, err = bundle.New()
			if err != nil {
				return bundle.Record{}, err
			}
		} else {
			return bundle.Record{}, mutationLoadError{err: err}
		}
	}
	timestamp := time.Now().UTC().Format(time.RFC3339)
	var appendOpts *bundle.AppendOptions
	if len(opts) > 0 {
		opt := opts[0]
		if opt.Timestamp == "" {
			opt.Timestamp = timestamp
		}
		appendOpts = &opt
	} else {
		appendOpts = &bundle.AppendOptions{
			Timestamp: timestamp,
		}
	}
	if err := b.AppendWithOptions(eventType, data, appendOpts); err != nil {
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
	os.Exit(runSnapshot(os.Args[2:], os.Stdout, os.Stderr))
}

func normalizeBundlePath(raw string) string {
	if raw == "" {
		return bundle.DefaultPath()
	}
	clean := filepath.Clean(raw)
	if info, err := os.Stat(clean); err == nil && info.IsDir() {
		return filepath.Join(clean, bundle.BundleFile) // #nosec G703 -- path is cleaned; bundle filename is constant
	}
	return clean
}

func parseVerifyArgs(args []string) (string, string, bool, bool, string, error) {
	path := ""
	format := verifyFormatText
	trace := false
	withAnchor := false
	rootsPath := ""
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--format":
			if i+1 >= len(args) {
				return "", "", false, false, "", fmt.Errorf("missing value for --format (expected text|json)")
			}
			format = strings.ToLower(strings.TrimSpace(args[i+1]))
			i++
		case strings.HasPrefix(arg, "--format="):
			format = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(arg, "--format=")))
		case arg == "--trace":
			trace = true
		case arg == "--with-anchor":
			withAnchor = true
		case arg == "--roots":
			if i+1 >= len(args) {
				return "", "", false, false, "", fmt.Errorf("missing value for --roots")
			}
			i++
			rootsPath = filepath.Clean(strings.TrimSpace(args[i]))
		case strings.HasPrefix(arg, "--roots="):
			rootsPath = filepath.Clean(strings.TrimSpace(strings.TrimPrefix(arg, "--roots=")))
		case strings.HasPrefix(arg, "--"):
			return "", "", false, false, "", fmt.Errorf("unknown flag %q", arg)
		default:
			if path != "" {
				return "", "", false, false, "", fmt.Errorf("verify accepts at most one bundle path")
			}
			path = normalizeBundlePath(arg)
		}
	}
	if path == "" {
		path = bundle.DefaultPath()
	}
	if format != verifyFormatText && format != verifyFormatJSON {
		return "", "", false, false, "", fmt.Errorf("invalid format %q (expected text|json)", format)
	}
	return path, format, trace, withAnchor, rootsPath, nil
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

func verifyWithTrace(b *bundle.Bundle, out io.Writer) error {
	prev := hash.GenesisHash
	hasManifest := len(b.Records) > 0 && b.Records[0].Event.Type == bundle.ManifestEventType
	for i, record := range b.Records {
		event := record.Event
		event.PrevHash = prev
		switch {
		case hasManifest && i == 0:
			event.Sequence = 0
		case hasManifest:
			event.Sequence = i
		default:
			event.Sequence = i + 1
		}

		computed, err := hash.Compute(event)
		if err != nil {
			return fmt.Errorf("hash: verify at index %d: %w", i, err)
		}

		match := computed == record.Hash
		fmt.Fprintf(
			out,
			"trace: event_index=%d seq=%d prev_hash=%s stored_hash=%s computed_hash=%s match=%t\n",
			i,
			event.Sequence,
			prev,
			record.Hash,
			computed,
			match,
		)
		if !match {
			return fmt.Errorf(
				"hash: verify: tamper detected at event %d (seq %d): expected %s, got %s",
				i,
				event.Sequence,
				record.Hash,
				computed,
			)
		}
		prev = computed
	}
	return nil
}

// cmdVerify verifies the integrity of the current bundle.
func cmdVerify() {
	path, outputFormat, trace, withAnchor, rootsPath, err := parseVerifyArgs(os.Args[2:])
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
	verifyErr := error(nil)
	if trace {
		verifyErr = verifyWithTrace(b, os.Stderr)
	} else {
		verifyErr = b.Verify()
	}
	if verifyErr != nil {
		if outputFormat == verifyFormatJSON {
			result := newVerifyResult(path, b, "invalid")
			result.Error = verifyErr.Error()
			printVerifyJSON(result)
		}
		fmt.Fprintf(os.Stderr, "✗ VERIFICATION FAILED: %v\n", verifyErr)
		os.Exit(exitIntegrityFailure)
	}
	if withAnchor {
		roots, err := loadVerifyRoots(rootsPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "✗ ANCHOR VERIFICATION FAILED: %v\n", err)
			os.Exit(exitIntegrityFailure)
		}
		prevRoots := verifyBundleAnchorRoots
		verifyBundleAnchorRoots = roots
		defer func() {
			verifyBundleAnchorRoots = prevRoots
		}()

		anchorOut := io.Writer(os.Stdout)
		if outputFormat == verifyFormatJSON {
			anchorOut = io.Discard
		}
		if err := verifyBundleAnchor(path, b, anchorOut); err != nil {
			if outputFormat == verifyFormatJSON {
				result := newVerifyResult(path, b, "invalid")
				result.Error = err.Error()
				printVerifyJSON(result)
			}
			fmt.Fprintf(os.Stderr, "✗ ANCHOR VERIFICATION FAILED: %v\n", err)
			os.Exit(exitIntegrityFailure)
		}
	}
	if outputFormat == verifyFormatJSON {
		printVerifyJSON(newVerifyResult(path, b, "valid"))
		return
	}
	fmt.Printf("✓ Bundle verified: %d events, chain intact.\n", len(b.Records))
}
