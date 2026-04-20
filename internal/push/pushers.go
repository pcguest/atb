package push

import (
	"context"
	"fmt"
	"time"
)

// Push is the transport-neutral bundle push interface.
type Push interface {
	Push(ctx context.Context, bundle []byte, meta PushMeta) error
}

// PushMeta carries the metadata sent alongside a pushed bundle.
type PushMeta struct {
	BundleID      string
	SealTimestamp time.Time
	ProfileID     string
	Digest        string
}

// S3Pusher uploads a bundle to an S3 object key using the existing WORM path.
type S3Pusher struct {
	Uploader  S3Uploader
	Bucket    string
	Key       string
	LockMode  string
	LockUntil string
}

// Push uploads bundle bytes to the configured S3 key.
func (p S3Pusher) Push(ctx context.Context, bundle []byte, meta PushMeta) error {
	_ = meta

	if p.Uploader == nil {
		return fmt.Errorf("s3 push: uploader is nil")
	}
	if p.Bucket == "" {
		return fmt.Errorf("s3 push: bucket is required")
	}
	if p.Key == "" {
		return fmt.Errorf("s3 push: key is required")
	}

	_, err := p.Uploader.PutObject(ctx, PutObjectInput{
		Bucket:    p.Bucket,
		Key:       p.Key,
		Body:      bundle,
		LockMode:  p.LockMode,
		LockUntil: p.LockUntil,
	})
	if err != nil {
		return fmt.Errorf("s3 push: %w", err)
	}
	return nil
}
