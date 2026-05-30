// SPDX-License-Identifier: MIT
package store

import (
	"context"
	"errors"
)

var (
	// ErrNotImplemented indicates the storage adapter is scaffold-only.
	ErrNotImplemented = errors.New("custos store: not implemented")
)

// WORMStore persists bundle artefacts in immutable storage.
type WORMStore interface {
	Put(ctx context.Context, receiptID string, bundle []byte) error
	Get(ctx context.Context, receiptID string) ([]byte, error)
}

// S3Adapter is a scaffold WORM store backed by S3 Object Lock.
type S3Adapter struct {
	Bucket string
	Prefix string
}

var _ WORMStore = (*S3Adapter)(nil)

// Put stores an opaque bundle artefact under the receipt ID key.
func (a *S3Adapter) Put(ctx context.Context, receiptID string, bundle []byte) error {
	_ = ctx
	_ = receiptID
	_ = bundle
	if a == nil {
		return ErrNotImplemented
	}
	return ErrNotImplemented
}

// Get retrieves a bundle artefact by receipt ID.
func (a *S3Adapter) Get(ctx context.Context, receiptID string) ([]byte, error) {
	_ = ctx
	_ = receiptID
	if a == nil {
		return nil, ErrNotImplemented
	}
	return nil, ErrNotImplemented
}
