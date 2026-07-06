// SPDX-License-Identifier: MIT
package main

import (
	"os"
	"strings"
)

func mortiseTokenFromEnv() (token, source string) {
	if token = strings.TrimSpace(os.Getenv("ATB_MORTISE_TOKEN")); token != "" {
		return token, "ATB_MORTISE_TOKEN"
	}
	if token = strings.TrimSpace(os.Getenv("ATB_CUSTOS_TOKEN")); token != "" {
		return token, "ATB_CUSTOS_TOKEN (deprecated)"
	}
	return "", ""
}
