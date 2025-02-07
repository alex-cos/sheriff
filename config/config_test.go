package config_test

import (
	"testing"

	"github.com/alex-cos/sheriff/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_Load(t *testing.T) {
	t.Parallel()

	t.Run("valid file", func(t *testing.T) {
		t.Parallel()

		var cfg config.Config
		err := cfg.Load("../testdata/valid.yaml")
		require.NoError(t, err)

		require.Len(t, cfg.Services, 3)

		assert.Equal(t, "database", cfg.Services[0].Name)
		assert.Equal(t, "postgres", cfg.Services[0].Command)
		assert.Equal(t, []string{"-D", "/var/lib/postgresql/data"}, cfg.Services[0].Argument)
		assert.Empty(t, cfg.Services[0].DependsOn)

		assert.Equal(t, "api", cfg.Services[1].Name)
		assert.Equal(t, []string{"database"}, cfg.Services[1].DependsOn)

		assert.Equal(t, "frontend", cfg.Services[2].Name)
	})

	t.Run("file not found", func(t *testing.T) {
		t.Parallel()

		var cfg config.Config
		err := cfg.Load("../testdata/nonexistent.yaml")
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to read configuration file")
	})

	t.Run("invalid YAML", func(t *testing.T) {
		t.Parallel()

		var cfg config.Config
		err := cfg.Load("../testdata/invalid.yaml")
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to decode configuration file")
	})

	t.Run("empty services", func(t *testing.T) {
		t.Parallel()

		var cfg config.Config
		err := cfg.Validate()
		require.Error(t, err)
		assert.ErrorContains(t, err, "at least one service")
	})

	t.Run("missing service name", func(t *testing.T) {
		t.Parallel()

		cfg := config.Config{
			Services: []config.ServiceConfig{
				{Name: "valid", Command: "echo"},
				{Name: "", Command: "ping"},
			},
		}
		err := cfg.Validate()
		require.Error(t, err)
		assert.ErrorContains(t, err, "non-empty name")
	})

	t.Run("empty service command", func(t *testing.T) {
		t.Parallel()

		cfg := config.Config{
			Services: []config.ServiceConfig{
				{Name: "foo", Command: ""},
			},
		}
		err := cfg.Validate()
		require.Error(t, err)
		assert.ErrorContains(t, err, "command is required")
	})

	t.Run("duplicate service name", func(t *testing.T) {
		t.Parallel()

		cfg := config.Config{
			Services: []config.ServiceConfig{
				{Name: "dup", Command: "echo"},
				{Name: "dup", Command: "ping"},
			},
		}
		err := cfg.Validate()
		require.Error(t, err)
		assert.ErrorContains(t, err, "duplicate")
	})

	t.Run("negative stopTimeout", func(t *testing.T) {
		t.Parallel()

		cfg := config.Config{
			StopTimeout: -1,
			Services:    []config.ServiceConfig{{Name: "s", Command: "echo"}},
		}
		err := cfg.Validate()
		require.Error(t, err)
		assert.ErrorContains(t, err, "stopTimeout")
	})

	t.Run("negative monitor period", func(t *testing.T) {
		t.Parallel()

		cfg := config.Config{
			Monitor:  config.MonitorConfig{Period: -5},
			Services: []config.ServiceConfig{{Name: "s", Command: "echo"}},
		}
		err := cfg.Validate()
		require.Error(t, err)
		assert.ErrorContains(t, err, "monitor.period")
	})

	t.Run("valid logLevel", func(t *testing.T) {
		t.Parallel()

		for _, lvl := range []string{"debug", "info", "warn", "error", ""} {
			cfg := config.Config{
				LogLevel: lvl,
				Services: []config.ServiceConfig{{Name: "s", Command: "echo"}},
			}
			assert.NoError(t, cfg.Validate(), "logLevel=%q", lvl)
		}
	})

	t.Run("invalid logLevel", func(t *testing.T) {
		t.Parallel()

		cfg := config.Config{
			LogLevel: "trace",
			Services: []config.ServiceConfig{{Name: "s", Command: "echo"}},
		}
		err := cfg.Validate()
		require.Error(t, err)
		assert.ErrorContains(t, err, "invalid logLevel")
	})
}
