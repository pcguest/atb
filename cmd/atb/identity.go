// SPDX-License-Identifier: MIT
package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pcguest/atb/internal/identity"
)

func cmdIdentity() {
	os.Exit(runIdentityCommand(os.Args[2:], os.Stdout, os.Stderr))
}

func runIdentityCommand(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printIdentityHelp(stderr)
		return exitUserError
	}
	switch args[0] {
	case "set":
		return runIdentitySet(args[1:], stdout, stderr)
	case "-h", "--help", "help":
		printIdentityHelp(stdout)
		return exitSuccess
	default:
		fmt.Fprintf(stderr, "atb identity: unknown sub-command %q\n", args[0])
		printIdentityHelp(stderr)
		return exitUserError
	}
}

func runIdentitySet(args []string, stdout, stderr io.Writer) int {
	var key, name, email string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--key":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "atb identity set: missing value for --key")
				return exitUserError
			}
			key = strings.TrimSpace(args[i+1])
			i++
		case "--name":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "atb identity set: missing value for --name")
				return exitUserError
			}
			name = strings.TrimSpace(args[i+1])
			i++
		case "--email":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "atb identity set: missing value for --email")
				return exitUserError
			}
			email = strings.TrimSpace(args[i+1])
			i++
		default:
			fmt.Fprintf(stderr, "atb identity set: unknown argument %q\n", args[i])
			return exitUserError
		}
	}
	path, err := identity.DefaultMapPath()
	if err != nil {
		fmt.Fprintf(stderr, "atb identity set: %v\n", err)
		return exitSystemError
	}
	if err := identity.WriteMapping(path, key, name, email, ""); err != nil {
		fmt.Fprintf(stderr, "atb identity set: %v\n", err)
		return exitSystemError
	}
	fmt.Fprintf(stdout, "identity mapping written to %s\n", path)
	return exitSuccess
}

func printIdentityHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: atb identity set --key <api-key> --name <display-name> [--email <email>]")
}
