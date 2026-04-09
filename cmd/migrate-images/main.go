package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	ctx := context.Background()

	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		log.Fatal("DATABASE_DSN is not set")
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	s3Endpoint := os.Getenv("S3_ENDPOINT_URL")
	s3Region := os.Getenv("S3_REGION_NAME")
	s3Bucket := os.Getenv("S3_BUCKET_NAME")
	s3AccessKey := os.Getenv("S3_ACCESS_KEY_ID")
	s3SecretKey := os.Getenv("S3_SECRET_ACCESS_KEY")

	if s3Endpoint == "" || s3Region == "" || s3Bucket == "" || s3AccessKey == "" || s3SecretKey == "" {
		log.Fatal("S3_ENDPOINT_URL, S3_REGION_NAME, S3_BUCKET_NAME, S3_ACCESS_KEY_ID, and S3_SECRET_ACCESS_KEY must all be set")
	}

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(s3Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(s3AccessKey, s3SecretKey, "")),
	)
	if err != nil {
		log.Fatalf("failed to load AWS config: %v", err)
	}

	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(s3Endpoint)
	})
	domain := fmt.Sprintf("%s/%s", s3Endpoint, s3Bucket)

	// Загружаем все файлы из static/img/ в S3
	files, err := filepath.Glob("static/img/*")
	if err != nil {
		log.Fatalf("failed to glob static/img: %v", err)
	}

	if len(files) == 0 {
		log.Println("no files found in static/img/")
		return
	}

	log.Printf("found %d files to upload", len(files))

	for _, localPath := range files {
		filename := filepath.Base(localPath)
		s3Key := fmt.Sprintf("ads/%s", filename)

		file, err := os.Open(localPath)
		if err != nil {
			log.Printf("SKIP: cannot open %s: %v", localPath, err)
			continue
		}

		_, err = s3Client.PutObject(ctx, &s3.PutObjectInput{
			Bucket: aws.String(s3Bucket),
			Key:    aws.String(s3Key),
			Body:   file,
			ACL:    types.ObjectCannedACLPublicRead,
		})
		file.Close()

		if err != nil {
			log.Printf("FAIL: upload %s: %v", s3Key, err)
			continue
		}

		newURL := fmt.Sprintf("%s/%s", domain, s3Key)
		oldPath := fmt.Sprintf("/static/img/%s", filename)

		// Обновляем БД: если путь ещё локальный — ставим S3 URL
		tag, err := pool.Exec(ctx,
			`UPDATE product_image SET file_path = $1 WHERE file_path = $2`,
			newURL, oldPath,
		)
		if err != nil {
			log.Printf("FAIL: update db for %s: %v", filename, err)
			continue
		}

		if tag.RowsAffected() > 0 {
			log.Printf("OK: %s -> %s (%d rows updated)", oldPath, newURL, tag.RowsAffected())
		} else {
			log.Printf("OK: uploaded %s (no DB rows to update)", s3Key)
		}
	}

	log.Println("migration complete")
}
