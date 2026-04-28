package s3

import (
	"context"
	"fmt"
	"mime/multipart"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	cfg "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/auth/internal/config"
	"github.com/google/uuid"
)

type Storage interface {
	UploadFile(ctx context.Context, file multipart.File, folder string, extension string) (string, error)
	DeleteFile(ctx context.Context, fileURL string) error
}

type Client struct {
	s3Client   *s3.Client
	bucketName string
	domain     string
}

func NewS3Client(ctx context.Context, s3Config cfg.S3Config) (Storage, error) {
	sdkCfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(s3Config.RegionName),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(s3Config.AccessKeyID, s3Config.SecretAccessKey, "")),
	)
	if err != nil {
		return nil, err
	}

	s3Client := s3.NewFromConfig(sdkCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(s3Config.EndpointURL)
	})

	return &Client{
		s3Client:   s3Client,
		bucketName: s3Config.BucketName,
		domain:     fmt.Sprintf("%s/%s", s3Config.EndpointURL, s3Config.BucketName),
	}, nil
}

func (c *Client) UploadFile(ctx context.Context, file multipart.File, folder string, extension string) (string, error) {
	fileName := fmt.Sprintf("%s/%s%s", folder, uuid.New().String(), extension)

	_, err := c.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(c.bucketName),
		Key:    aws.String(fileName),
		Body:   file,
		ACL:    types.ObjectCannedACLPublicRead,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file to s3: %w", err)
	}

	return fmt.Sprintf("%s/%s", c.domain, fileName), nil
}

func (c *Client) DeleteFile(ctx context.Context, fileURL string) error {
	prefix := c.domain + "/"
	if !strings.HasPrefix(fileURL, prefix) {
		return fmt.Errorf("invalid file URL: does not belong to this bucket")
	}
	key := strings.TrimPrefix(fileURL, prefix)
	if key == "" {
		return fmt.Errorf("invalid file URL: empty key")
	}

	_, err := c.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete file from s3: %w", err)
	}

	return nil
}
