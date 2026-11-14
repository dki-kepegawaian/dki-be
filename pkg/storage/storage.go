// Package storage menyimpan berkas di object storage S3-compatible (Garage).
package storage

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Storage interface {
	Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
	Get(ctx context.Context, key string) (io.ReadCloser, int64, error)
	Delete(ctx context.Context, key string) error
}

type s3Storage struct {
	client *minio.Client
	bucket string
}

// NewFromEnv membaca S3_ENDPOINT (host:port), S3_ACCESS_KEY, S3_SECRET_KEY,
// S3_BUCKET, S3_REGION (default "garage"), dan S3_USE_SSL ("true"/"false").
func NewFromEnv() (Storage, error) {
	endpoint, bucket := os.Getenv("S3_ENDPOINT"), os.Getenv("S3_BUCKET")
	if endpoint == "" || bucket == "" {
		return nil, fmt.Errorf("S3_ENDPOINT dan S3_BUCKET wajib diisi")
	}
	region := os.Getenv("S3_REGION")
	if region == "" {
		region = "garage"
	}
	client, err := minio.New(endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(os.Getenv("S3_ACCESS_KEY"), os.Getenv("S3_SECRET_KEY"), ""),
		Secure:       os.Getenv("S3_USE_SSL") == "true",
		Region:       region,
		BucketLookup: minio.BucketLookupPath, // Garage tanpa root_domain memakai path-style
	})
	if err != nil {
		return nil, err
	}
	return &s3Storage{client: client, bucket: bucket}, nil
}

func (s *s3Storage) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, r, size, minio.PutObjectOptions{ContentType: contentType})
	return err
}

func (s *s3Storage) Get(ctx context.Context, key string) (io.ReadCloser, int64, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, 0, err
	}
	info, err := obj.Stat()
	if err != nil {
		obj.Close()
		return nil, 0, err
	}
	return obj, info.Size, nil
}

func (s *s3Storage) Delete(ctx context.Context, key string) error {
	return s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
}
