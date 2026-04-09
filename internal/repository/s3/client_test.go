package s3

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeFile реализует multipart.File для тестов.
type fakeFile struct {
	*bytes.Reader
}

func (f *fakeFile) Close() error { return nil }

// mockS3API реализует S3API для тестов.
type mockS3API struct {
	putObjectFn    func(ctx context.Context, input *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	deleteObjectFn func(ctx context.Context, input *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
}

func (m *mockS3API) PutObject(ctx context.Context, input *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	return m.putObjectFn(ctx, input, optFns...)
}

func (m *mockS3API) DeleteObject(ctx context.Context, input *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	return m.deleteObjectFn(ctx, input, optFns...)
}

func newTestClient(mock *mockS3API) *Client {
	return &Client{
		s3Client:   mock,
		bucketName: "test-bucket",
		domain:     "https://s3.example.com/test-bucket",
	}
}

func TestClient_UploadFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		folder    string
		extension string
		putErr    error
		wantErr   bool
	}{
		{
			name:      "success",
			folder:    "avatars",
			extension: ".jpg",
			putErr:    nil,
			wantErr:   false,
		},
		{
			name:      "s3 put error",
			folder:    "avatars",
			extension: ".png",
			putErr:    errors.New("access denied"),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var capturedInput *s3.PutObjectInput
			mock := &mockS3API{
				putObjectFn: func(_ context.Context, input *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
					capturedInput = input
					return &s3.PutObjectOutput{}, tt.putErr
				},
			}
			client := newTestClient(mock)
			file := &fakeFile{bytes.NewReader([]byte("file content"))}

			url, err := client.UploadFile(context.Background(), file, tt.folder, tt.extension)

			if tt.wantErr {
				require.Error(t, err)
				assert.Empty(t, url)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, "test-bucket", *capturedInput.Bucket)
			assert.True(t, strings.HasPrefix(*capturedInput.Key, tt.folder+"/"))
			assert.True(t, strings.HasSuffix(*capturedInput.Key, tt.extension))
			assert.True(t, strings.HasPrefix(url, client.domain+"/"+tt.folder+"/"))
			assert.True(t, strings.HasSuffix(url, tt.extension))
		})
	}
}

func TestClient_DeleteFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		fileURL   string
		deleteErr error
		wantKey   string
		wantErr   bool
	}{
		{
			name:      "success",
			fileURL:   "https://s3.example.com/test-bucket/avatars/photo.jpg",
			deleteErr: nil,
			wantKey:   "avatars/photo.jpg",
			wantErr:   false,
		},
		{
			name:      "s3 delete error",
			fileURL:   "https://s3.example.com/test-bucket/ads/img.png",
			deleteErr: errors.New("not found"),
			wantKey:   "ads/img.png",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var capturedInput *s3.DeleteObjectInput
			mock := &mockS3API{
				deleteObjectFn: func(_ context.Context, input *s3.DeleteObjectInput, _ ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
					capturedInput = input
					return &s3.DeleteObjectOutput{}, tt.deleteErr
				},
			}
			client := newTestClient(mock)

			err := client.DeleteFile(context.Background(), tt.fileURL)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, "test-bucket", *capturedInput.Bucket)
			assert.Equal(t, tt.wantKey, *capturedInput.Key)
		})
	}
}
