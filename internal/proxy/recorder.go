// SPDX-License-Identifier: MIT
package proxy

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
)

// BundleRecorder appends capture events to a local ATB bundle.
type BundleRecorder struct {
	path          string
	mortisePusher MortisePusherInterface
	mu            sync.Mutex
}

// NewBundleRecorder returns a recorder for the given bundle path.
func NewBundleRecorder(path string, mortisePusher MortisePusherInterface) *BundleRecorder {
	return &BundleRecorder{path: path, mortisePusher: mortisePusher}
}

// AppendEvent appends a canonical event to the configured bundle.
func (r *BundleRecorder) AppendEvent(ev *event.Event) error {
	_, err := r.AppendEventHash(ev)
	return err
}

// AppendEventHash appends an event and returns the bundle record hash.
func (r *BundleRecorder) AppendEventHash(ev *event.Event) (string, error) {
	if r == nil || ev == nil {
		return "", errors.New("proxy: nil recorder or event")
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	// Append to local bundle
	recordHash, err := r.appendEventLocked(ev)
	if err != nil {
		return "", err
	}

	// Mortise ingests whole bundles, not individual events. The completed
	// bundle is pushed once on session close (see sessionCloseCallback).

	return recordHash, nil
}

// AppendSessionClose writes an atb.session.close summary event.
func (r *BundleRecorder) AppendSessionClose(sess *Session) error {
	// Fixed: AppendSessionClose now reuses the snapshot path and discards bytes for no-push callers.
	_, err := r.appendSessionCloseAndSnapshot(sess)
	return err
}

// Fixed: appendSessionCloseAndSnapshot reads bundle bytes under the recorder lock for async push safety.
func (r *BundleRecorder) appendSessionCloseAndSnapshot(sess *Session) ([]byte, error) {
	if r == nil || sess == nil {
		return nil, nil
	}
	ev := &event.Event{
		Type:      TypeSessionClose,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Data:      SessionCloseRecord(sess),
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, err := r.appendEventLocked(ev); err != nil {
		return nil, err
	}
	// Fixed: The snapshot is read while no other recorder append can change the bundle file.
	snapshot, err := os.ReadFile(r.path)
	if err != nil {
		return nil, fmt.Errorf("proxy: read bundle snapshot for Mortise push: %w", err)
	}
	return snapshot, nil
}

// sessionCloseCallback is the callback function for SessionManager to close a session.
func (r *BundleRecorder) sessionCloseCallback(sess *Session) error {
	// Fixed: Nil sessions should not reach the async push branch after a no-op append.
	if sess == nil {
		return nil
	}
	// Fixed: The session close append returns the immutable bytes pushed asynchronously.
	snapshot, err := r.appendSessionCloseAndSnapshot(sess)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error appending session close event for session %s: %v\n", sess.ID, err)
		return err
	}

	if r.mortisePusher != nil {
		// Capture the session ID before the goroutine so logging does not
		// depend on later session mutation. The locked snapshot is pushed,
		// not a re-read of a changing bundle file.
		sessionID := sess.ID
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()
			receipt, err := r.mortisePusher.PushBundle(ctx, snapshot)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error pushing bundle %s to Mortise for session %s: %v\n", r.path, sessionID, err)
				return
			}
			if receipt != nil {
				fmt.Fprintf(os.Stderr, "Mortise receipt %s for session %s (bundle hash %s)\n", receipt.ReceiptID, sessionID, receipt.BundleHash)
			}
		}()
	}
	return nil
}

func (r *BundleRecorder) appendEventLocked(ev *event.Event) (string, error) {
	b, err := loadBundleForAppend(r.path)
	if err != nil {
		return "", err
	}
	opts := &bundle.AppendOptions{Timestamp: ev.Timestamp}
	if ev.ActorID != nil {
		opts.ActorID = ev.ActorID
	}
	if err := b.AppendWithOptions(ev.Type, ev.Data, opts); err != nil {
		return "", err
	}
	if len(b.Records) == 0 {
		return "", errors.New("proxy: append produced no bundle records")
	}
	recordHash := b.Records[len(b.Records)-1].Hash
	if err := b.Save(r.path); err != nil {
		return "", err
	}
	return recordHash, nil
}

func loadBundleForAppend(path string) (*bundle.Bundle, error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return bundle.New()
		}
		return nil, err
	}
	return bundle.LoadVerified(path)
}
