package config

import (
	"flag"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMustLoadConfig_SuccessEnv(t *testing.T) {
	// this subtest verifies string duration parsing (preferred format)
	t.Run("string values", func(t *testing.T) {
		os.Args = []string{"cmd"}
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
		content := `{
			"env": "local",
			"token_ttl": "1h",
			"refresh_ttl": "168h",
			"http": {"port": 8080},
			"cleanup_interval": "1m"
		}`
		tmpFile, err := os.CreateTemp("", "config-*.json")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		_, err = tmpFile.Write([]byte(content))
		require.NoError(t, err)
		err = tmpFile.Close()
		require.NoError(t, err)

		os.Setenv("CONFIG_PATH", tmpFile.Name())
		defer os.Unsetenv("CONFIG_PATH")

		t.Setenv("CONFIG_PATH", tmpFile.Name())
		t.Setenv("TOKEN_SECRET", "test-secret-key")
		t.Setenv("DATABASE_DSN", "postgres://user:pass@localhost:5432/testdb?sslmode=disable")
		t.Setenv("REDIS_ADDR", "localhost:6379")
		t.Setenv("S3_ENDPOINT_URL", "https://hb.vkcs.cloud")
		t.Setenv("S3_REGION_NAME", "ru-msk")
		t.Setenv("S3_BUCKET_NAME", "test-bucket")
		t.Setenv("S3_ACCESS_KEY_ID", "test-key")
		t.Setenv("S3_SECRET_ACCESS_KEY", "test-secret")

		cfg := MustLoadConfig()
		assert.NotNil(t, cfg)
		assert.Equal(t, "test-secret-key", cfg.TokenSecret)
		assert.Equal(t, "local", cfg.Env)
		assert.Equal(t, 8080, cfg.HTTP.Port)
		assert.Equal(t, time.Hour, cfg.TokenTTL)
		assert.Equal(t, 168*time.Hour, cfg.RefreshTTL)
		assert.Equal(t, time.Minute, cfg.CleanupInterval)
	})
}

func TestMustLoadConfig_EmptyPath(t *testing.T) {
	os.Args = []string{"cmd"}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	t.Setenv("CONFIG_PATH", "")
	t.Setenv("TOKEN_SECRET", "dummy")
	t.Setenv("DATABASE_DSN", "postgres://user:pass@localhost:5432/testdb?sslmode=disable")
	assert.PanicsWithValue(t, "config path is empty", func() {
		MustLoadConfig()
	})
}

func TestMustLoadConfig_NotExist(t *testing.T) {
	os.Args = []string{"cmd"}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	os.Setenv("CONFIG_PATH", "non_existent_file.json")
	defer os.Unsetenv("CONFIG_PATH")

	assert.Panics(t, func() {
		MustLoadConfig()
	})
}
