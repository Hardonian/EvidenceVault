package email

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestLogSender_Send(t *testing.T) {
	var buf bytes.Buffer

	handler := slog.NewTextHandler(&buf, nil)
	logger := slog.New(handler)

	originalLogger := slog.Default()
	defer slog.SetDefault(originalLogger)

	slog.SetDefault(logger)

	sender := LogSender{}
	err := sender.Send("test@example.com", "Test Subject", "Test Body")

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	logOutput := buf.String()

	if !strings.Contains(logOutput, "email_send") {
		t.Errorf("expected log output to contain 'email_send', got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "to=test@example.com") {
		t.Errorf("expected log output to contain 'to=test@example.com', got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "subject=\"Test Subject\"") {
		t.Errorf("expected log output to contain 'subject=\"Test Subject\"', got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "body=\"Test Body\"") {
		t.Errorf("expected log output to contain 'body=\"Test Body\"', got: %s", logOutput)
	}
}
