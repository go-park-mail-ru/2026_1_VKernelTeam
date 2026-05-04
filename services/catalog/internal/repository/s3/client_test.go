package s3

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/catalog/internal/repository/s3/mocks"
	"github.com/golang/mock/gomock"
)

func newClient(api S3API) *Client {
	return &Client{s3Client: api, bucketName: "test-bucket", domain: "https://hb.vkcs.cloud/test-bucket"}
}

func newAPI(t *testing.T) *mocks.MockS3API {
	t.Helper()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	return mocks.NewMockS3API(ctrl)
}

func TestUploadFile_Success(t *testing.T) {
	api := newAPI(t)
	api.EXPECT().PutObject(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, in *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
			if *in.Bucket != "test-bucket" {
				t.Errorf("bucket mismatch: %s", *in.Bucket)
			}
			if !strings.HasPrefix(*in.Key, "ads/") || !strings.HasSuffix(*in.Key, ".jpg") {
				t.Errorf("key mismatch: %s", *in.Key)
			}
			return &s3.PutObjectOutput{}, nil
		})

	c := newClient(api)
	url, err := c.UploadFile(context.Background(), nil, "ads", ".jpg")
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if !strings.HasPrefix(url, "https://hb.vkcs.cloud/test-bucket/ads/") || !strings.HasSuffix(url, ".jpg") {
		t.Fatalf("unexpected url: %s", url)
	}
}

func TestUploadFile_Error(t *testing.T) {
	api := newAPI(t)
	api.EXPECT().PutObject(gomock.Any(), gomock.Any()).Return(nil, errors.New("boom"))

	c := newClient(api)
	if _, err := c.UploadFile(context.Background(), nil, "ads", ".jpg"); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDeleteFile_InvalidURL(t *testing.T) {
	c := newClient(newAPI(t))
	if err := c.DeleteFile(context.Background(), "https://other.bucket/foo.jpg"); err == nil {
		t.Fatal("expected error for foreign URL")
	}
}

func TestDeleteFile_Success(t *testing.T) {
	api := newAPI(t)
	api.EXPECT().DeleteObject(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, in *s3.DeleteObjectInput, _ ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
			if *in.Key != "ads/abc.jpg" {
				t.Errorf("key mismatch: %s", *in.Key)
			}
			return &s3.DeleteObjectOutput{}, nil
		})

	c := newClient(api)
	if err := c.DeleteFile(context.Background(), "https://hb.vkcs.cloud/test-bucket/ads/abc.jpg"); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestDeleteFile_S3Error(t *testing.T) {
	api := newAPI(t)
	api.EXPECT().DeleteObject(gomock.Any(), gomock.Any()).Return(nil, errors.New("boom"))

	c := newClient(api)
	if err := c.DeleteFile(context.Background(), "https://hb.vkcs.cloud/test-bucket/ads/abc.jpg"); err == nil {
		t.Fatal("expected error, got nil")
	}
}
