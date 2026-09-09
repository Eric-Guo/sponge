package initial

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-dev-frame/sponge/internal/config"
)

func clearThrusterEnvironment(t *testing.T) {
	t.Helper()
	for _, name := range []string{"GZIP_COMPRESSION_ENABLED", "GZIP_COMPRESSION_DISABLE_ON_AUTH", "GZIP_COMPRESSION_JITTER", "FORWARD_HEADERS"} {
		for _, prefix := range []string{"", "THRUSTER_"} {
			key := prefix + name
			t.Setenv(key, "") // restore the caller's value at cleanup
			require.NoError(t, os.Unsetenv(key))
		}
	}
}

func TestLoadConfigCompression(t *testing.T) {
	clearThrusterEnvironment(t)
	for _, tt := range []struct {
		name, yaml string
		jitter     int
	}{
		{"default", "http:\n  gzipEnabled: true\n", 32},
		{"disabled", "http:\n  gzipJitter: 0\n", 0},
		{"custom", "http:\n  gzipJitter: 64\n", 64},
	} {
		t.Run(tt.name, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), tt.name+".yml")
			require.NoError(t, os.WriteFile(file, []byte(tt.yaml), 0600))
			require.NoError(t, loadConfig(file))
			require.Equal(t, tt.jitter, config.Get().HTTP.GzipJitter)
		})
	}
	require.Error(t, loadConfig(filepath.Join(t.TempDir(), "missing.yml")))
}

func TestLoadConfigEnvironmentPrecedence(t *testing.T) {
	clearThrusterEnvironment(t)
	t.Setenv("GZIP_COMPRESSION_JITTER", "64")
	t.Setenv("THRUSTER_GZIP_COMPRESSION_JITTER", "0")
	t.Setenv("GZIP_COMPRESSION_DISABLE_ON_AUTH", "false")
	t.Setenv("THRUSTER_GZIP_COMPRESSION_DISABLE_ON_AUTH", "true")
	t.Setenv("GZIP_COMPRESSION_ENABLED", "1")
	t.Setenv("THRUSTER_FORWARD_HEADERS", "false")
	file := filepath.Join(t.TempDir(), "service.yml")
	require.NoError(t, os.WriteFile(file, []byte("http:\n  gzipEnabled: false\n  gzipJitter: 32\nproxy:\n  forwardHeaders: true\n"), 0600))
	require.NoError(t, loadConfig(file))
	cfg := config.Get()
	require.Zero(t, cfg.HTTP.GzipJitter)
	require.True(t, cfg.HTTP.GzipDisableOnAuth)
	require.True(t, cfg.HTTP.GzipEnabled)
	require.False(t, cfg.Proxy.ForwardHeaders)

	for _, name := range []string{"GZIP_COMPRESSION_ENABLED", "GZIP_COMPRESSION_DISABLE_ON_AUTH", "GZIP_COMPRESSION_JITTER", "FORWARD_HEADERS"} {
		t.Run(name, func(t *testing.T) {
			t.Setenv("THRUSTER_"+name, "invalid")
			require.ErrorContains(t, loadConfig(file), name)
			require.Same(t, cfg, config.Get(), "invalid configuration must not replace the active configuration")
		})
	}
}
