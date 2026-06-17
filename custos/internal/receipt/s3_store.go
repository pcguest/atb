package receipt

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// S3ReceiptStore stores receipt JSON documents in an S3 bucket.
type S3ReceiptStore struct {
	client *s3.Client
	bucket string
	prefix string // Optional prefix for all objects (e.g., "receipts/v1/")
	Policy RetentionPolicy
}

// NewS3ReceiptStore creates an S3-backed receipt store.
func NewS3ReceiptStore(ctx context.Context, bucket, region, endpoint string, policy RetentionPolicy) (*S3ReceiptStore, error) {
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

	return &S3ReceiptStore{
		client: client,
		bucket: bucket,
		prefix: "v1/", // Hardcode version for now, similar to filesystem
		Policy: policy,
	}, nil
}

// StoreReceipt stores a receipt.
func (s *S3ReceiptStore) StoreReceipt(ctx context.Context, receipt Receipt) error {
	return s.Save(ctx, receipt)
}

// Save writes a receipt JSON file to S3.
func (s *S3ReceiptStore) Save(ctx context.Context, receipt Receipt) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(receipt.ReceiptID) == "" {
		return errors.New("receipt store: receipt id required")
	}

	key := s.receiptKey(receipt.ReceiptID)
	data, err := json.Marshal(receipt)
	if err != nil {
		return fmt.Errorf("receipt store: marshal receipt: %w", err)
	}

	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
	})
	if err != nil {
		return fmt.Errorf("receipt store: put object to S3: %w", err)
	}
	return nil
}

// GetReceipt retrieves a receipt.
func (s *S3ReceiptStore) GetReceipt(ctx context.Context, receiptID string) (Receipt, error) {
	return s.Get(ctx, receiptID)
}

// Get retrieves one receipt JSON file by receipt ID from S3.
func (s *S3ReceiptStore) Get(ctx context.Context, receiptID string) (Receipt, error) {
	if err := ctx.Err(); err != nil {
		return Receipt{}, err
	}

	key := s.receiptKey(receiptID)
	output, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var noSuchKey *types.NoSuchKey
		if errors.As(err, &noSuchKey) {
			return Receipt{}, fmt.Errorf("%w: %s", ErrReceiptNotFound, receiptID)
		}
		return Receipt{}, fmt.Errorf("receipt store: get object from S3: %w", err)
	}
	defer output.Body.Close()

	data, err := io.ReadAll(output.Body)
	if err != nil {
		return Receipt{}, fmt.Errorf("receipt store: read object body from S3: %w", err)
	}

	var receipt Receipt
	if err := json.Unmarshal(data, &receipt); err != nil {
		return Receipt{}, fmt.Errorf("receipt store: unmarshal receipt: %w", err)
	}
	return receipt, nil
}

// List returns all readable receipt JSON files sorted by SubmittedAt ascending.
func (s *S3ReceiptStore) List(ctx context.Context) ([]Receipt, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	var allReceipts []Receipt
	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(s.prefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("receipt store: list objects from S3: %w", err)
		}
		for _, obj := range page.Contents {
			receiptID := strings.TrimSuffix(strings.TrimPrefix(*obj.Key, s.prefix), ".json")
			receipt, err := s.Get(ctx, receiptID)
			if err != nil {
				log.Printf("receipt store: skip unreadable or malformed S3 receipt %s: %v", *obj.Key, err)
				continue
			}
			allReceipts = append(allReceipts, receipt)
		}
	}

	sort.Slice(allReceipts, func(i, j int) bool {
		return allReceipts[i].SubmittedAt.Before(allReceipts[j].SubmittedAt)
	})
	return allReceipts, nil
}

// CleanUp applies the configured retention policy to the stored receipts.
func (s *S3ReceiptStore) CleanUp(ctx context.Context) error {
	if s.Policy.MaxAgeDays == 0 && s.Policy.MaxCount == 0 {
		return nil // No retention policy configured
	}

	receipts, err := s.List(ctx)
	if err != nil {
		return fmt.Errorf("receipt store: cleanup list receipts: %w", err)
	}

	// Apply MaxAgeDays policy
	if s.Policy.MaxAgeDays > 0 {
		cutoff := time.Now().Add(time.Duration(-s.Policy.MaxAgeDays) * 24 * time.Hour)
		for _, receipt := range receipts {
			if receipt.SubmittedAt.Before(cutoff) {
				if err := s.deleteReceipt(ctx, receipt.ReceiptID); err != nil {
					log.Printf("receipt store: cleanup failed to delete old S3 receipt %s: %v", receipt.ReceiptID, err)
				}
			}
		}
		// Re-list after age-based deletion to ensure accurate count for MaxCount policy
		receipts, err = s.List(ctx)
		if err != nil {
			return fmt.Errorf("receipt store: cleanup re-list S3 receipts after age cleanup: %w", err)
		}
	}

	// Apply MaxCount policy
	if s.Policy.MaxCount > 0 && len(receipts) > s.Policy.MaxCount {
		// Receipts are already sorted by SubmittedAt ascending from List()
		toDelete := len(receipts) - s.Policy.MaxCount
		for i := 0; i < toDelete; i++ {
			if err := s.deleteReceipt(ctx, receipts[i].ReceiptID); err != nil {
				log.Printf("receipt store: cleanup failed to delete excess S3 receipt %s: %v", receipts[i].ReceiptID, err)
			}
		}
	}
	return nil
}

// deleteReceipt removes a receipt file from the store.
func (s *S3ReceiptStore) deleteReceipt(ctx context.Context, receiptID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	key := s.receiptKey(receiptID)
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("receipt store: delete object from S3: %w", err)
	}
	return nil
}

func (s *S3ReceiptStore) receiptKey(receiptID string) string {
	return s.prefix + receiptID + ".json"
}
