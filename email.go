// email_sender.go

package main

import (
	"context"
	"encoding/base64"

	"google.golang.org/api/gmail/v1"
)

type EmailSender interface {
	SendEmail(ctx context.Context, to, subject, body string) error
}

type GmailEmailSender struct {
	service *gmail.Service
}

func NewGmailEmailSender(service *gmail.Service) *GmailEmailSender {
	return &GmailEmailSender{service: service}
}

func (s *GmailEmailSender) SendEmail(ctx context.Context, to, subject, body string) error {
	// Compose the email message
	message := "To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"\r\n" + body

	// Send the email
	_, err := s.service.Users.Messages.Send("me", &gmail.Message{
		Raw: base64.URLEncoding.EncodeToString([]byte(message)),
	}).Do()

	return err
}
