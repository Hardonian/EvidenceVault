package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Client interface {
	Upload(ctx context.Context, key string, body io.Reader) (string, error)
}

type LocalClient struct{ BasePath string }

func (l LocalClient) Upload(_ context.Context, key string, body io.Reader) (string, error) {
	if err := os.MkdirAll(l.BasePath, 0o755); err != nil {
		return "", err
	}
	cleanBase := filepath.Clean(l.BasePath)
	path := filepath.Join(cleanBase, key)
	// Prevent path traversal: resolved path must remain within BasePath
	if !strings.HasPrefix(filepath.Clean(path), cleanBase+string(filepath.Separator)) && filepath.Clean(path) != cleanBase {
		return "", fmt.Errorf("path traversal denied: %s", key)
	}
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := io.Copy(f, body); err != nil {
		return "", err
	}
	return fmt.Sprintf("local://%s", key), nil
}
