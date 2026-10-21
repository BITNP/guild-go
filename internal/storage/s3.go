package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
)

// S3Store stores objects in a private S3-compatible bucket.
type S3Store struct {
	client *s3.Client
	bucket string
}

// NewS3 creates an S3 store. Region and path-style addressing are constants.
func NewS3(ctx context.Context, endpoint, bucket, accessKey, secretKey string) (*S3Store, error) {
	if endpoint == "" || bucket == "" || accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("S3 configuration is incomplete")
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(s3Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("load S3 config: %w", err)
	}
	client := s3.NewFromConfig(cfg, func(options *s3.Options) {
		options.BaseEndpoint = aws.String(endpoint)
		options.UsePathStyle = s3ForcePathStyle
	})
	return &S3Store{client: client, bucket: bucket}, nil
}

// Put uploads one object.
func (s *S3Store) Put(ctx context.Context, key string, data []byte, contentType string) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(key),
		Body:          bytes.NewReader(data),
		ContentLength: aws.Int64(int64(len(data))),
		ContentType:   aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("put S3 object: %w", err)
	}
	return nil
}

// Get returns an object stream and metadata.
func (s *S3Store) Get(ctx context.Context, key string) (io.ReadCloser, string, int64, error) {
	output, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isNotFound(err) {
			return nil, "", 0, ErrNotFound
		}
		return nil, "", 0, fmt.Errorf("get S3 object: %w", err)
	}
	contentType := ""
	if output.ContentType != nil {
		contentType = normalizeContentType(*output.ContentType)
	}
	size := int64(0)
	if output.ContentLength != nil {
		size = *output.ContentLength
	}
	return output.Body, contentType, size, nil
}

// Delete removes one object.
func (s *S3Store) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("delete S3 object: %w", err)
	}
	return nil
}

func isNotFound(err error) bool {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		return apiErr.ErrorCode() == "NoSuchKey" || apiErr.ErrorCode() == "NotFound"
	}
	return false
}

// TimeoutForUpload is a safe context timeout for small image uploads.
const TimeoutForUpload = 30 * time.Second
