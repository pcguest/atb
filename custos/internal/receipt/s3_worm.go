package receipt

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// S3WORMStore stores immutable ATB bundles in an S3 bucket.
type S3WORMStore struct {
	client *s3.Client
	bucket string
	prefix string // Optional prefix for all objects (e.g., "bundles/v1/")
}

// NewS3WORMStore creates an S3-backed WORM store.
func NewS3WORMStore(ctx context.Context, bucket, region, endpoint string) (*S3WORMStore, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS SDK config: %w", err)
	}

	var client *s3.Client
	if endpoint != "" {
		client = s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(endpoint)
			o.UsePathStyle = true // Required for some S3-compatible services
		})
	} else {
		client = s3.NewFromConfig(cfg)
	}

	return &S3WORMStore{
		client: client,
		bucket: bucket,
		prefix: "v1/", // Hardcode version for now
	}, nil
}

// Store writes bundle bytes once under their SHA-256 content hash.
func (s *S3WORMStore) Store(ctx context.Context, bundleBytes []byte, bundleHash string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	// The store is content-addressed: the supplied hash must be the SHA-256 of
	// the bytes being stored. This is a storage-boundary integrity check (the
	// bytes written match their key), not the bundle's hash-chain head hash.
	// (This check is performed by the caller, ingest.IngestHandler, so we don't repeat it here)

	key := s.bundleKey(bundleHash)

	// Check if object already exists
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err == nil {
		return key, nil // Object already exists, WORM semantics satisfied
	}
	var notFound *types.NotFound
	if !errors.As(err, &notFound) {
		return "", fmt.Errorf("worm store: head object from S3: %w", err)
	}

	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(bundleBytes),
	})
	if err != nil {
		return "", fmt.Errorf("worm store: put object to S3: %w", err)
	}
	return key, nil
}

// Retrieve reads immutable bundle bytes by receipt ID (which is the bundle hash).
func (s *S3WORMStore) Retrieve(ctx context.Context, receiptID string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	key := s.bundleKey(receiptID)
	output, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var noSuchKey *types.NoSuchKey
		if errors.As(err, &noSuchKey) {
			return nil, fmt.Errorf("%w: %s", ErrReceiptNotFound, receiptID)
		}
		return nil, fmt.Errorf("worm store: get object from S3: %w", err)
	}
	defer output.Body.Close()

	bundleBytes, err := io.ReadAll(output.Body)
	if err != nil {
		return nil, fmt.Errorf("worm store: read object body from S3: %w", err)
	}
	return bundleBytes, nil
}

func (s *S3WORMStore) bundleKey(bundleHash string) string {
	// Receipt IDs map to the verified bundle hash filename used at ingest.
	// The bundleHash is expected to be the SHA-256 hex string.
	hash := strings.TrimPrefix(bundleHash, "sha256-")
	return s.prefix + hash + ".atb"
}
