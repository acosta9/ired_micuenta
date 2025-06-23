package utils

import (
	"os"
	"strconv"

	"gopkg.in/gomail.v2"
)

func SendEmail(to []string, subject string, body string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", "Besser Solutions <norespuesta@bessersolutions.com>")
	m.SetHeader("To", to...)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	m.Embed("public/assets/logo_mail.png", gomail.SetHeader(map[string][]string{"Content-ID": {"<image001>"}}))

	smtpServer := os.Getenv("SMTP_SERVER")
	smtpEmail := os.Getenv("SMTP_EMAIL")
	smtpPassword := os.Getenv("SMTP_PASSWORD")
	smtpPort, err := strconv.Atoi(os.Getenv("SMTP_PORT"))
	if err != nil {
		Logline("error parsing smtp_port", err)
		return err
	}

	d := gomail.NewDialer(smtpServer, smtpPort, smtpEmail, smtpPassword)

	if err := d.DialAndSend(m); err != nil {
		return err
	}
	return nil
}
