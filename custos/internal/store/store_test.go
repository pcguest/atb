// SPDX-License-Identifier: MIT
package store_test

import (
	"context"
	"testing"

	"github.com/pcguest/custos/internal/store"
)

func TestS3AdapterStub(t *testing.T) {
	adapter := &store.S3Adapter{Bucket: "demo", Prefix: "custos/"}
	if err := adapter.Put(context.Background(), "sha256-deadbeef", []byte("bundle")); err == nil {
		t.Fatal("expected stub error")
	}
}
