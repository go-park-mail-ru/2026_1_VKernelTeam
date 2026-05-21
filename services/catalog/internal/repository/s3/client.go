package s3

//go:generate mockgen -source=client.go -destination=mocks/mock_client.go -package=mocks

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
	cfg "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/catalog/internal/config"
	"github.com/google/uuid"
)

// Storage описывает контракт объектного хранилища для загрузки и удаления файлов.
type Storage interface {
	UploadFile(ctx context.Context, file multipart.File, folder string, extension string) (string, error)
	DeleteFile(ctx context.Context, fileURL string) error
}

// S3API описывает подмножество AWS S3 SDK, используемое клиентом (для подмены в тестах).
type S3API interface {
	PutObject(ctx context.Context, input *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	DeleteObject(ctx context.Context, input *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
}

// Client — реализация Storage поверх AWS S3 SDK.
type Client struct {
	s3Client   S3API
	bucketName string
	domain     string
}

// NewS3Client создаёт клиент S3 на основе переданной конфигурации.
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

// UploadFile загружает файл в указанную папку бакета и возвращает его публичный URL.
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

// DeleteFile удаляет файл из бакета по его публичному URL.
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
