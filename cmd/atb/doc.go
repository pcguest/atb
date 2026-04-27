// SPDX-License-Identifier: MIT
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	apiv1 "github.com/pcguest/atb/pkg/api/v1"
)

const defaultOpenAPIOutputPath = "docs/api/openapi.yaml"

func cmdDoc() {
	if len(os.Args) < 3 {
		printDocUsage()
		os.Exit(exitUserError)
	}

	sub := strings.ToLower(strings.TrimSpace(os.Args[2]))
	switch sub {
	case "gen-openapi":
		if err := runDocGenOpenAPI(os.Args[3:]); err != nil {
			fmt.Fprintf(os.Stderr, "atb doc gen-openapi: %v\n", err)
			os.Exit(exitUserError)
		}
	default:
		fmt.Fprintf(os.Stderr, "atb doc: unknown subcommand %q\n", sub)
		printDocUsage()
		os.Exit(exitUserError)
	}
}

func runDocGenOpenAPI(args []string) error {
	outputPath := defaultOpenAPIOutputPath

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--output":
			if i+1 >= len(args) {
				return fmt.Errorf("missing value for --output")
			}
			outputPath = strings.TrimSpace(args[i+1])
			i++
		case strings.HasPrefix(arg, "--output="):
			outputPath = strings.TrimSpace(strings.TrimPrefix(arg, "--output="))
		case strings.HasPrefix(arg, "-"):
			return fmt.Errorf("unknown flag %q", arg)
		default:
			return fmt.Errorf("unexpected argument %q", arg)
		}
	}

	if strings.TrimSpace(outputPath) == "" {
		return fmt.Errorf("--output cannot be empty")
	}

	spec := apiv1.OpenAPITemplate()
	if len(spec) == 0 {
		return fmt.Errorf("openapi template is empty")
	}
	trimmed := strings.TrimSpace(string(spec))
	if !strings.HasPrefix(trimmed, "openapi:") {
		return fmt.Errorf("openapi template missing required 'openapi:' header")
	}
	if spec[len(spec)-1] != '\n' {
		spec = append(spec, '\n')
	}

	cleanOutput := filepath.Clean(outputPath)
	if err := os.MkdirAll(filepath.Dir(cleanOutput), 0750); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	if err := os.WriteFile(cleanOutput, spec, 0600); err != nil {
		return fmt.Errorf("write openapi output: %w", err)
	}

	fmt.Printf("✓ Generated OpenAPI YAML: %s\n", cleanOutput)
	return nil
}

func printDocUsage() {
	fmt.Println("Usage: atb doc gen-openapi [--output docs/api/openapi.yaml]")
}
