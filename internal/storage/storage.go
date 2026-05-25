package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type Client interface {
	Upload(ctx context.Context, key string, body io.Reader) (string, error)
}

type LocalClient struct{ BasePath string }

func (l LocalClient) Upload(_ context.Context, key string, body io.Reader) (string, error) {
	if err := os.MkdirAll(l.BasePath, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(l.BasePath, key)
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
