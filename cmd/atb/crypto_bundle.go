package main

import (
	"encoding/json"
	"fmt"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/canonicalize"
	"github.com/pcguest/atb/internal/hash"
)

type encryptedBundlePayload struct {
	HeadHash string          `json:"head_hash"`
	Records  []bundle.Record `json:"records"`
}

func canonicalPayloadFromBundle(b *bundle.Bundle) ([]byte, error) {
	if b == nil {
		return nil, fmt.Errorf("bundle is nil")
	}
	head := hash.GenesisHash
	if len(b.Records) > 0 {
		head = b.Records[len(b.Records)-1].Hash
	}
	payload := encryptedBundlePayload{
		HeadHash: head,
		Records:  b.Records,
	}
	canonical, err := canonicalize.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("canonicalize payload: %w", err)
	}
	return canonical, nil
}

func bundleFromCanonicalPayload(raw []byte) (*bundle.Bundle, error) {
	var payload encryptedBundlePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("parse decrypted payload: %w", err)
	}
	b := bundle.New()
	b.Records = append(b.Records, payload.Records...)

	// Post-decryption integrity check: recompute the chain and ensure head hash matches metadata.
	if err := b.Verify(); err != nil {
		return nil, fmt.Errorf("verify decrypted payload chain: %w", err)
	}
	recomputedHead := hash.GenesisHash
	if len(b.Records) > 0 {
		recomputedHead = b.Records[len(b.Records)-1].Hash
	}
	if payload.HeadHash != recomputedHead {
		return nil, fmt.Errorf("verify decrypted payload head hash: expected %s, got %s", payload.HeadHash, recomputedHead)
	}
	return b, nil
}
