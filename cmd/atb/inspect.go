// SPDX-License-Identifier: MIT
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/pcguest/atb/internal/bundle"
)

var errInspectHelp = errors.New("inspect help requested")

type inspectConfig struct {
	BundlePath string
	JSON       bool
	Seq        int
	SeqSet     bool
}

func cmdInspect() {
	os.Exit(runInspect(os.Args[2:], os.Stdout, os.Stderr))
}

func runInspect(args []string, stdout, stderr io.Writer) int {
	cfg, err := parseInspectCommandArgs(args)
	if err != nil {
		if errors.Is(err, errInspectHelp) {
			printInspectCommandUsage(stdout)
			return exitSuccess
		}
		fmt.Fprintf(stderr, "atb inspect: %v\n", err)
		printInspectCommandUsage(stderr)
		return exitUserError
	}

	b, err := bundle.Load(cfg.BundlePath)
	if err != nil {
		fmt.Fprintf(stderr, "atb inspect: %v\n", err)
		return exitUserError
	}

	if cfg.SeqSet {
		data, err := inspectRecordData(b, cfg.Seq)
		if err != nil {
			fmt.Fprintf(stderr, "atb inspect: %v\n", err)
			return exitUserError
		}
		if _, err := fmt.Fprintln(stdout, string(escapeJSONUnsafeRunes(data))); err != nil {
			fmt.Fprintf(stderr, "atb inspect: write output: %v\n", err)
			return exitSystemError
		}
		return exitSuccess
	}

	if cfg.JSON {
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		enc.SetIndent("", "  ")
		if err := enc.Encode(b.Records); err != nil {
			fmt.Fprintf(stderr, "atb inspect: encode json output: %v\n", err)
			return exitSystemError
		}
		if _, err := stdout.Write(escapeJSONUnsafeRunes(buf.Bytes())); err != nil {
			fmt.Fprintf(stderr, "atb inspect: write output: %v\n", err)
			return exitSystemError
		}
		return exitSuccess
	}

	if err := renderInspectTable(stdout, b); err != nil {
		fmt.Fprintf(stderr, "atb inspect: write output: %v\n", err)
		return exitSystemError
	}
	return exitSuccess
}

func parseInspectCommandArgs(args []string) (inspectConfig, error) {
	cfg := inspectConfig{
		BundlePath: bundle.DefaultPath(),
	}
	bundlePathSet := false

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-h" || arg == "--help":
			return cfg, errInspectHelp
		case arg == "--bundle" || arg == "-b":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for %s", arg)
			}
			if bundlePathSet {
				return cfg, fmt.Errorf("bundle path already set")
			}
			i++
			cfg.BundlePath = normalizeBundlePath(args[i])
			bundlePathSet = true
		case strings.HasPrefix(arg, "--bundle="):
			if bundlePathSet {
				return cfg, fmt.Errorf("bundle path already set")
			}
			cfg.BundlePath = normalizeBundlePath(strings.TrimPrefix(arg, "--bundle="))
			bundlePathSet = true
		case arg == "--json":
			cfg.JSON = true
		case arg == "--seq":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for --seq")
			}
			i++
			seq, err := strconv.Atoi(strings.TrimSpace(args[i]))
			if err != nil {
				return cfg, fmt.Errorf("invalid value for --seq %q", args[i])
			}
			cfg.Seq = seq
			cfg.SeqSet = true
		case strings.HasPrefix(arg, "--seq="):
			seq, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(arg, "--seq=")))
			if err != nil {
				return cfg, fmt.Errorf("invalid value for --seq %q", strings.TrimPrefix(arg, "--seq="))
			}
			cfg.Seq = seq
			cfg.SeqSet = true
		case strings.HasPrefix(arg, "-"):
			return cfg, fmt.Errorf("unknown flag %q", arg)
		default:
			if bundlePathSet {
				return cfg, fmt.Errorf("expected at most one bundle path")
			}
			cfg.BundlePath = normalizeBundlePath(arg)
			bundlePathSet = true
		}
	}

	return cfg, nil
}

func printInspectCommandUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: atb inspect [bundle_path] [--bundle path/to/file.atb] [--json] [--seq <n>]")
}

func renderInspectTable(w io.Writer, b *bundle.Bundle) error {
	useANSI := os.Getenv("NO_COLOR") == "" && isTerminal(w)

	header := "SEQ  TYPE                     TIMESTAMP             DATA (first 80 chars)"
	divider := "---  -----------------------  --------------------  --------------------------------"
	if useANSI {
		header = ansiBold + header + ansiReset
		divider = ansiBold + divider + ansiReset
	}

	if _, err := fmt.Fprintln(w, header); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, divider); err != nil {
		return err
	}
	if manifest := b.Manifest(); manifest != nil && manifest.CaptureRunID != "" {
		if _, err := fmt.Fprintf(w, "Capture run: %s\n", sanitizeForTerminal(manifest.CaptureRunID)); err != nil {
			return err
		}
	}

	for _, record := range b.Records {
		if _, err := fmt.Fprintf(
			w,
			"%3d  %-23s  %-20s  %s\n",
			record.Event.Sequence,
			sanitizeForTerminal(record.Event.Type),
			sanitizeForTerminal(record.Event.Timestamp),
			inspectDataPreview(record.Event.Data, 80),
		); err != nil {
			return err
		}
	}

	return nil
}

func inspectRecordData(b *bundle.Bundle, seq int) ([]byte, error) {
	for _, record := range b.Records {
		if record.Event.Sequence != seq {
			continue
		}
		return marshalInspectData(record.Event.Data)
	}
	return nil, fmt.Errorf("seq %d out of range (bundle has %d records)", seq, len(b.Records))
}

func marshalInspectData(data any) ([]byte, error) {
	if raw, ok := data.(string); ok && json.Valid([]byte(raw)) {
		var payload any
		if err := json.Unmarshal([]byte(raw), &payload); err != nil {
			return nil, fmt.Errorf("decode record data: %w", err)
		}
		return json.MarshalIndent(payload, "", "  ")
	}
	return json.MarshalIndent(data, "", "  ")
}

func inspectDataPreview(data any, limit int) string {
	preview := inspectDataString(data)
	preview = strings.Join(strings.Fields(preview), " ")
	preview = sanitizeForTerminal(preview)

	runes := []rune(preview)
	if len(runes) <= limit {
		return preview
	}
	if limit <= 3 {
		return string(runes[:limit])
	}
	return string(runes[:limit-3]) + "..."
}

// sanitizeForTerminal replaces ASCII control characters (C0 including ESC), C1
// controls, and Unicode line/paragraph separators and bidi overrides with a
// visible escape, so attacker-controlled evidence cannot inject terminal escape
// sequences or spoof rendered output.
func sanitizeForTerminal(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r == '\t':
			b.WriteByte(' ')
		case r < 0x20 || (r >= 0x7f && r <= 0x9f):
			fmt.Fprintf(&b, "\\x%02x", r)
		case r == 0x2028 || r == 0x2029,
			r >= 0x202a && r <= 0x202e,
			r >= 0x2066 && r <= 0x2069,
			r == 0x061c,
			r >= 0x200b && r <= 0x200f,
			r == 0x2060 || r == 0xfeff:
			fmt.Fprintf(&b, "\\u%04x", r)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// escapeJSONUnsafeRunes rewrites bidi and zero-width control runes that
// encoding/json emits raw into their \uXXXX escapes. The result is still valid
// JSON (parsing yields the identical string) but cannot visually spoof a
// terminal rendering the output.
func escapeJSONUnsafeRunes(b []byte) []byte {
	var out bytes.Buffer
	for i := 0; i < len(b); {
		r, size := utf8.DecodeRune(b[i:])
		if r != utf8.RuneError && isBidiOrZeroWidth(r) {
			fmt.Fprintf(&out, "\\u%04x", r)
		} else {
			out.Write(b[i : i+size])
		}
		i += size
	}
	return out.Bytes()
}

func isBidiOrZeroWidth(r rune) bool {
	switch {
	case r == 0x2028 || r == 0x2029:
		return true
	case r >= 0x202a && r <= 0x202e:
		return true
	case r >= 0x2066 && r <= 0x2069:
		return true
	case r == 0x061c:
		return true
	case r >= 0x200b && r <= 0x200f:
		return true
	case r == 0x2060 || r == 0xfeff:
		return true
	}
	return false
}

func inspectDataString(data any) string {
	if raw, ok := data.(string); ok {
		if json.Valid([]byte(raw)) {
			var compact bytes.Buffer
			if err := json.Compact(&compact, []byte(raw)); err == nil {
				return compact.String()
			}
		}
		return raw
	}

	encoded, err := json.Marshal(data)
	if err != nil {
		return fmt.Sprintf("%v", data)
	}
	return string(encoded)
}
