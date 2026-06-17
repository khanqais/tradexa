package utils

import (
	"fmt"
	"net/smtp"
	"os"
)

func SendEmail(to, subject, htmlBody string) error {
	from := os.Getenv("SMTP_EMAIL")
	password := os.Getenv("SMTP_PASSWORD")
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")

	if from == "" || password == "" || smtpHost == "" || smtpPort == "" {
		return fmt.Errorf("SMTP environment variables are not configured (SMTP_EMAIL, SMTP_PASSWORD, SMTP_HOST, SMTP_PORT)")
	}

	auth := smtp.PlainAuth("", from, password, smtpHost)

	msg := []byte("To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		"\r\n" +
		htmlBody + "\r\n")

	addr := smtpHost + ":" + smtpPort
	return smtp.SendMail(addr, auth, from, []string{to}, msg)
}
