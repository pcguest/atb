package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	anchorpkg "github.com/pcguest/atb/internal/anchor"
	"github.com/pcguest/atb/internal/bundle"
)

type anchorConfig struct {
	BundlePath string
	TSAURL     string
}

type anchorResult struct {
	BundlePath    string
	TokenPath     string
	TSAURL        string
	CertifiedTime string
	BundleHash    string
	TSRHash       string
	EventData     string
}

type anchorEventData struct {
	TSAURL        string `json:"tsa_url"`
	BundleHash    string `json:"bundle_hash"`
	TSRHash       string `json:"tsr_hash"`
	CertifiedTime string `json:"certified_time"`
}

func parseAnchorArgs(args []string) (anchorConfig, error) {
	cfg := anchorConfig{
		BundlePath: bundle.DefaultPath(),
		TSAURL:     anchorpkg.DefaultTSAURL,
	}
	bundlePathSet := false

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--tsa-url":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for --tsa-url")
			}
			cfg.TSAURL = strings.TrimSpace(args[i+1])
			i++
		case strings.HasPrefix(arg, "--tsa-url="):
			cfg.TSAURL = strings.TrimSpace(strings.TrimPrefix(arg, "--tsa-url="))
		case strings.HasPrefix(arg, "--"):
			return cfg, fmt.Errorf("unknown flag %q", arg)
		default:
			if bundlePathSet {
				return cfg, fmt.Errorf("anchor accepts at most one bundle path")
			}
			cfg.BundlePath = normalizeBundlePath(arg)
			bundlePathSet = true
		}
	}

	if cfg.TSAURL == "" {
		return cfg, fmt.Errorf("--tsa-url cannot be empty")
	}
	return cfg, nil
}

func runAnchor(cfg anchorConfig) (anchorResult, error) {
	result := anchorResult{
		BundlePath: cfg.BundlePath,
		TSAURL:     cfg.TSAURL,
	}

	b, err := bundle.Load(cfg.BundlePath)
	if err != nil {
		return result, err
	}

	bundleHash, err := anchorpkg.HashBundle(cfg.BundlePath)
	if err != nil {
		return result, err
	}
	result.BundleHash = hex.EncodeToString(bundleHash)

	tsrBytes, err := anchorpkg.Request(cfg.TSAURL, bundleHash)
	if err != nil {
		return result, err
	}

	result.TokenPath = cfg.BundlePath + ".tsr"
	if err := os.WriteFile(result.TokenPath, tsrBytes, 0600); err != nil {
		return result, fmt.Errorf("write anchor token: %w", err)
	}

	certifiedTime, err := anchorpkg.ParseGenTime(tsrBytes)
	if err != nil {
		return result, err
	}
	result.CertifiedTime = certifiedTime

	tsrHash := sha256.Sum256(tsrBytes)
	result.TSRHash = hex.EncodeToString(tsrHash[:])
	result.EventData = fmt.Sprintf(
		`{"tsa_url":%q,"bundle_hash":%q,"tsr_hash":%q,"certified_time":%q}`,
		cfg.TSAURL,
		result.BundleHash,
		result.TSRHash,
		certifiedTime,
	)

	if err := b.AppendWithOptions(bundle.AnchorEventType, result.EventData, &bundle.AppendOptions{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		return result, err
	}
	if err := b.Save(cfg.BundlePath); err != nil {
		return result, fmt.Errorf("save: %w", err)
	}

	return result, nil
}

func cmdAnchor() {
	cfg, err := parseAnchorArgs(os.Args[2:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "atb anchor: %v\n", err)
		fmt.Fprintln(os.Stderr, "Usage: atb anchor [bundle_path] [--tsa-url <url>]")
		os.Exit(exitUserError)
	}

	result, err := runAnchor(cfg)
	if err != nil {
		exitCode := exitSystemError
		if errors.Is(err, os.ErrNotExist) || errors.Is(err, os.ErrPermission) {
			exitCode = classifyBundleLoadError(err)
		} else if strings.Contains(strings.ToLower(err.Error()), "unmarshal") || strings.Contains(strings.ToLower(err.Error()), "scan") {
			exitCode = classifyBundleLoadError(err)
		}
		fmt.Fprintf(os.Stderr, "atb anchor: %v\n", err)
		os.Exit(exitCode)
	}

	fmt.Printf("Anchored. TSA: %s  Certified: %s  Token: %s\n", result.TSAURL, result.CertifiedTime, result.TokenPath)
}

func verifyBundleAnchor(bundlePath string, b *bundle.Bundle, out io.Writer) error {
	anchorIndex, data, found, err := latestAnchorEventData(b)
	if err != nil {
		return err
	}
	if !found {
		fmt.Fprintln(out, "No anchor event found in bundle — skipping anchor verification")
		return nil
	}

	tokenPath := bundlePath + ".tsr"
	tokenBytes, err := os.ReadFile(tokenPath) // #nosec G304 -- derived from the verified bundle path
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(out, "No anchor token found at %s — skipping anchor verification\n", tokenPath)
			return nil
		}
		return fmt.Errorf("read anchor token: %w", err)
	}

	tokenHash := sha256.Sum256(tokenBytes)
	if got := hex.EncodeToString(tokenHash[:]); got != data.TSRHash {
		return fmt.Errorf("anchor token hash mismatch: event=%s file=%s", data.TSRHash, got)
	}

	tokenTime, err := anchorpkg.ParseGenTime(tokenBytes)
	if err != nil {
		return fmt.Errorf("parse anchor token genTime: %w", err)
	}
	if tokenTime != data.CertifiedTime {
		return fmt.Errorf("anchor certified time mismatch: event=%s token=%s", data.CertifiedTime, tokenTime)
	}

	snapshotHash, err := hashBundleSnapshotBeforeAnchor(b, anchorIndex)
	if err != nil {
		return err
	}
	if got := hex.EncodeToString(snapshotHash); got != data.BundleHash {
		return fmt.Errorf("anchor bundle hash mismatch: event=%s snapshot=%s", data.BundleHash, got)
	}

	fmt.Fprintf(out, "Anchor verified. Certified: %s\n", data.CertifiedTime)
	return nil
}

func latestAnchorEventData(b *bundle.Bundle) (int, anchorEventData, bool, error) {
	for i := len(b.Records) - 1; i >= 0; i-- {
		if b.Records[i].Event.Type != bundle.AnchorEventType {
			continue
		}

		raw, ok := b.Records[i].Event.Data.(string)
		if !ok {
			return 0, anchorEventData{}, false, fmt.Errorf("anchor event data must be a JSON string")
		}

		var data anchorEventData
		if err := json.Unmarshal([]byte(raw), &data); err != nil {
			return 0, anchorEventData{}, false, fmt.Errorf("parse anchor event data: %w", err)
		}
		return i, data, true, nil
	}
	return -1, anchorEventData{}, false, nil
}

func hashBundleSnapshotBeforeAnchor(b *bundle.Bundle, anchorIndex int) ([]byte, error) {
	snapshot := &bundle.Bundle{
		Records: append([]bundle.Record(nil), b.Records[:anchorIndex]...),
	}

	tmp, err := os.CreateTemp("", "atb-anchor-*.atb")
	if err != nil {
		return nil, fmt.Errorf("create anchor snapshot temp file: %w", err)
	}
	tmpPath := tmp.Name()
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return nil, fmt.Errorf("close anchor snapshot temp file: %w", err)
	}
	defer os.Remove(tmpPath)

	if err := snapshot.Save(tmpPath); err != nil {
		return nil, fmt.Errorf("save anchor snapshot temp file: %w", err)
	}
	return anchorpkg.HashBundle(tmpPath)
}
