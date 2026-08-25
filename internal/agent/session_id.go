// SPDX-License-Identifier: MIT
package agent

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

func newSessionID() (SessionID, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", fmt.Errorf("agent: generate session id: %w", err)
	}
	return SessionID("sess_" + hex.EncodeToString(buf[:])), nil
}
