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
	// this subtest exercises numeric duration encoding (legacy behaviour)
	t.Run("numeric values", func(t *testing.T) {
		os.Args = []string{"cmd"}
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
		content := `{
			"env": "local",
			"storage_path": "./data/storage.db",
			"token_ttl": 3600000000000,
			"http": {"port": 8080},
			"cleanup_interval": 60000000000,
			"token_secret": "my-secret"
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

		cfg := MustLoadConfig()
		assert.NotNil(t, cfg)
		assert.Equal(t, "local", cfg.Env)
		assert.Equal(t, "./data/storage.db", cfg.StoragePath)
		assert.Equal(t, 8080, cfg.HTTP.Port)
		assert.Equal(t, "my-secret", cfg.TokenSecret)
		assert.Equal(t, time.Hour, cfg.TokenTTL)
		assert.Equal(t, time.Minute, cfg.CleanupInterval)
	})

	// this subtest verifies string duration parsing (preferred format)
	t.Run("string values", func(t *testing.T) {
		os.Args = []string{"cmd"}
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
		content := `{
			"env": "local",
			"storage_path": "./data/storage.db",
			"token_ttl": "1h",
			"http": {"port": 8080},
			"cleanup_interval": "1m",
			"token_secret": "my-secret"
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

		cfg := MustLoadConfig()
		assert.NotNil(t, cfg)
		assert.Equal(t, "local", cfg.Env)
		assert.Equal(t, "./data/storage.db", cfg.StoragePath)
		assert.Equal(t, 8080, cfg.HTTP.Port)
		assert.Equal(t, "my-secret", cfg.TokenSecret)
		assert.Equal(t, time.Hour, cfg.TokenTTL)
		assert.Equal(t, time.Minute, cfg.CleanupInterval)
	})
}

func TestMustLoadConfig_EmptyPath(t *testing.T) {
	os.Args = []string{"cmd"}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	os.Setenv("CONFIG_PATH", "")
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
