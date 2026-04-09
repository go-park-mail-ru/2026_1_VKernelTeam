package s3

import (
	"context"
	"fmt"
	"mime/multipart"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	cfg "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/config"
	"github.com/google/uuid"
)

type Storage interface {
	UploadFile(ctx context.Context, file multipart.File, folder string, extension string) (string, error)
	DeleteFile(ctx context.Context, fileURL string) error
}

type Uploader interface {
	UploadObject(ctx context.Context, input *transfermanager.UploadObjectInput, opts ...func(*transfermanager.Options)) (*transfermanager.UploadObjectOutput, error)
}

type Deleter interface {
	DeleteObject(ctx context.Context, input *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
}

type Client struct {
	uploader   Uploader
	deleter    Deleter
	bucketName string
	domain     string
}

func NewS3Client(ctx context.Context, s3Config cfg.S3Config) (Storage, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(s3Config.RegionName),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(s3Config.AccessKeyID, s3Config.SecretAccessKey, "")),
	)
	if err != nil {
		return nil, err
	}

	// Переопределяем Endpoint для VK Cloud
	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(s3Config.EndpointURL)
	})

	return &Client{
		uploader:   transfermanager.New(s3Client),
		deleter:    s3Client,
		bucketName: s3Config.BucketName,
		domain:     fmt.Sprintf("%s/%s", s3Config.EndpointURL, s3Config.BucketName),
	}, nil
}

func (c *Client) UploadFile(ctx context.Context, file multipart.File, folder string, extension string) (string, error) {
	fileName := fmt.Sprintf("%s/%s%s", folder, uuid.New().String(), extension)

	_, err := c.uploader.UploadObject(ctx, &transfermanager.UploadObjectInput{
		Bucket: aws.String(c.bucketName),
		Key:    aws.String(fileName),
		Body:   file,
	})

	if err != nil {
		return "", fmt.Errorf("failed to upload file to s3: %w", err)
	}

	return fmt.Sprintf("%s/%s", c.domain, fileName), nil
}

func (c *Client) DeleteFile(ctx context.Context, fileURL string) error {
	// Извлекаем ключ из URL
	key := fileURL[len(c.domain)+1:]

	_, err := c.deleter.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucketName),
		Key:    aws.String(key),
	})

	if err != nil {
		return fmt.Errorf("failed to delete file from s3: %w", err)
	}

	return nil
}
