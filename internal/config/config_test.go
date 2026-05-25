package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("CRON_SECRET", "secret")
	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, 10, cfg.FreeTierLimit)
	os.Unsetenv("DATABASE_URL")
}
