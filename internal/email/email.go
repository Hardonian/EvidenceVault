package email

import "log/slog"

type Sender interface {
	Send(to, subject, body string) error
}

type LogSender struct{}

func (LogSender) Send(to, subject, body string) error {
	slog.Info("email_send", "to", to, "subject", subject, "body", body)
	return nil
}
