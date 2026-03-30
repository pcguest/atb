package main

import (
	"fmt"
	"io"
	"os"
)

func cmdBundle() {
	os.Exit(runBundle(os.Args[2:], os.Stdout, os.Stderr))
}

func runBundle(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "atb bundle: missing sub-command")
		printBundleCommandUsage(stderr)
		return exitUserError
	}

	switch args[0] {
	case "new":
		return runInit(args[1:], stdout, stderr)
	case "-h", "--help", "help":
		printBundleCommandUsage(stdout)
		return exitSuccess
	default:
		fmt.Fprintf(stderr, "atb bundle: unknown sub-command %q\n", args[0])
		printBundleCommandUsage(stderr)
		return exitUserError
	}
}

func printBundleCommandUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: atb bundle new [--dry-run] [--format text|json]")
}
