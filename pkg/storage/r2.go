package storage

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"valancis-backend/pkg/utils"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type R2Storage struct {
	client        *s3.Client
	bucketName    string
	publicURL     string
	uploadTimeout time.Duration
}

func NewR2Storage(ctx context.Context, accountId, accessKey, secretKey, bucketName, publicURL string, uploadTimeout time.Duration) (*R2Storage, error) {
	r2Resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL: fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountId),
		}, nil
	})

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithEndpointResolverWithOptions(r2Resolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		config.WithRegion("auto"),
	)
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	return &R2Storage{
		client:        client,
		bucketName:    bucketName,
		publicURL:     strings.TrimSuffix(publicURL, "/"),
		uploadTimeout: uploadTimeout,
	}, nil
}

func (s *R2Storage) UploadFile(ctx context.Context, file multipart.File, fileHeader *multipart.FileHeader) (string, error) {
	// Generate unique filename
	ext := filepath.Ext(fileHeader.Filename)
	filename := fmt.Sprintf("uploads/%s%s", utils.GenerateUUID(), ext) // Organized in folder

	// Create context with upload timeout
	uploadCtx, cancel := context.WithTimeout(ctx, s.uploadTimeout)
	defer cancel()

	// Upload to R2 with timeout protection
	_, err := s.client.PutObject(uploadCtx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(filename),
		Body:        file,
		ContentType: aws.String(fileHeader.Header.Get("Content-Type")),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload to R2: %w", err)
	}

	// Return Public URL
	return fmt.Sprintf("%s/%s", s.publicURL, filename), nil
}

// UploadBuffer uploads a byte slice as a file (used for processed images)
func (s *R2Storage) UploadBuffer(ctx context.Context, data []byte, contentType string) (string, error) {
	// 1. Determine Extension from Content-Type
	ext := ".bin"
	switch contentType {
	case "image/webp":
		ext = ".webp"
	case "image/jpeg":
		ext = ".jpg"
	case "image/png":
		ext = ".png"
	}

	// 2. Generate Filename
	filename := fmt.Sprintf("uploads/%s%s", utils.GenerateUUID(), ext)

	// 3. Create Reader
	reader := bytes.NewReader(data)

	// 4. Create context
	uploadCtx, cancel := context.WithTimeout(ctx, s.uploadTimeout)
	defer cancel()

	// 5. Upload
	_, err := s.client.PutObject(uploadCtx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(filename),
		Body:        reader,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload buffer to R2: %w", err)
	}

	return fmt.Sprintf("%s/%s", s.publicURL, filename), nil
}

// DeleteFile deletes a file from R2/S3 by its full URL
func (s *R2Storage) DeleteFile(ctx context.Context, fileURL string) error {
	// 1. Extract Key from URL
	// URL format: https://pub-xxx.r2.dev/uploads/filename.webp
	// Key format: uploads/filename.webp
	// Simple strategy: Split by last occurrence of publicUrl
	// If the URL doesn't contain publicURL, assume it might be a relative path or handle gracefully

	var key string
	if strings.HasPrefix(fileURL, s.publicURL) {
		key = strings.TrimPrefix(fileURL, s.publicURL)
		key = strings.TrimPrefix(key, "/") // Remove leading slash
	} else {
		// Fallback: assume the last part is the key or it's a relative path
		// If it doesn't match our domain, strictly we might not want to delete it,
		// but let's assume valid key if passed here.
		// Safe bet: just return if it doesn't match our host to avoid deleting "other" things?
		// For now, let's assume the caller passes a valid URL that belongs to us.
		return fmt.Errorf("invalid file URL: domain mismatch")
	}

	if key == "" {
		return fmt.Errorf("invalid file key derived from URL")
	}

	// 2. Delete from S3
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete file from R2: %w", err)
	}

	return nil
}
