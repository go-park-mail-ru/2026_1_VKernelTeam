package s3

import (
	"context"
	"testing"

	cfg "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/auth/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newClient(t *testing.T) *Client {
	t.Helper()
	storage, err := NewS3Client(context.Background(), cfg.S3Config{
		EndpointURL:     "http://localhost:9000",
		RegionName:      "us-east-1",
		BucketName:      "bucket",
		AccessKeyID:     "key",
		SecretAccessKey: "secret",
	})
	require.NoError(t, err)
	c, ok := storage.(*Client)
	require.True(t, ok)
	return c
}

func TestNewS3Client_OK(t *testing.T) {
	c := newClient(t)
	assert.Equal(t, "bucket", c.bucketName)
	assert.Equal(t, "http://localhost:9000/bucket", c.domain)
	assert.NotNil(t, c.s3Client)
}

func TestDeleteFile_InvalidPrefix(t *testing.T) {
	c := newClient(t)
	err := c.DeleteFile(context.Background(), "https://other.host/bucket/folder/file.png")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not belong to this bucket")
}

func TestDeleteFile_EmptyKey(t *testing.T) {
	c := newClient(t)
	err := c.DeleteFile(context.Background(), "http://localhost:9000/bucket/")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty key")
}
