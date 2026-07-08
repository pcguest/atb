// SPDX-License-Identifier: MIT
package auth

import (
	"context"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwk"
)

func TestJWTRefreshStopsWhenLifecycleContextIsCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	validator := &JWTValidator{logger: slog.Default()}
	var calls atomic.Int32
	firstCall := make(chan struct{}, 1)

	validator.startRefresh(ctx, time.Millisecond, func(context.Context) (jwk.Set, error) {
		calls.Add(1)
		select {
		case firstCall <- struct{}{}:
		default:
		}
		return jwk.NewSet(), nil
	})

	select {
	case <-firstCall:
	case <-time.After(time.Second):
		t.Fatal("refresh did not run")
	}
	cancel()
	time.Sleep(20 * time.Millisecond)
	afterCancel := calls.Load()
	time.Sleep(20 * time.Millisecond)
	if got := calls.Load(); got != afterCancel {
		t.Fatalf("refresh continued after cancellation: calls advanced from %d to %d", afterCancel, got)
	}
}
