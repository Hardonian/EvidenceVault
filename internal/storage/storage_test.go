package storage

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalClient_Upload(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		tempDir := t.TempDir()
		client := LocalClient{BasePath: tempDir}

		key := "test-file.txt"
		content := "hello world"

		url, err := client.Upload(context.Background(), key, strings.NewReader(content))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expectedURL := "local://test-file.txt"
		if url != expectedURL {
			t.Errorf("expected URL %q, got %q", expectedURL, url)
		}

		// Verify file contents
		path := filepath.Join(tempDir, key)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read written file: %v", err)
		}
		if string(data) != content {
			t.Errorf("expected file content %q, got %q", content, string(data))
		}
	})

	t.Run("mkdir_error", func(t *testing.T) {
		tempDir := t.TempDir()
		// Create a file where we want our base path to be, so MkdirAll fails
		badBasePath := filepath.Join(tempDir, "file-not-dir")
		if err := os.WriteFile(badBasePath, []byte("blocker"), 0644); err != nil {
			t.Fatalf("setup failed: %v", err)
		}

		client := LocalClient{BasePath: badBasePath}
		_, err := client.Upload(context.Background(), "test.txt", strings.NewReader("hello"))
		if err == nil {
			t.Fatal("expected error when MkdirAll fails, got nil")
		}
	})

	t.Run("create_file_error", func(t *testing.T) {
		tempDir := t.TempDir()
		client := LocalClient{BasePath: tempDir}

		// passing a nested key where the directory doesn't exist
		// os.Create will fail because the folder "nested" does not exist
		_, err := client.Upload(context.Background(), "nested/test.txt", strings.NewReader("hello"))
		if err == nil {
			t.Fatal("expected error when os.Create fails, got nil")
		}
	})

	t.Run("io_copy_error", func(t *testing.T) {
		tempDir := t.TempDir()
		client := LocalClient{BasePath: tempDir}

		// A reader that returns an error
		badReader := &errorReader{}
		_, err := client.Upload(context.Background(), "test.txt", badReader)
		if err == nil {
			t.Fatal("expected error during io.Copy, got nil")
		}
	})
}

type errorReader struct{}

func (r *errorReader) Read(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF
}
