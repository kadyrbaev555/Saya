package email

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"time"
)

type Sender struct{ Address, From, Username, Password string }

func (s Sender) SendRegistrationCode(ctx context.Context, to, code string) error {
	host, port, err := net.SplitHostPort(s.Address)
	if err != nil {
		return err
	}
	from, err := mail.ParseAddress(s.From)
	if err != nil {
		return err
	}
	conn, err := (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, "tcp", s.Address)
	if err != nil {
		return err
	}
	defer conn.Close()
	stop := context.AfterFunc(ctx, func() { conn.Close() })
	defer stop()
	if err = conn.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return err
	}
	config := &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}
	if port == "465" {
		connTLS := tls.Client(conn, config)
		return s.deliver(connTLS, host, config, false, from.Address, to, code)
	}
	return s.deliver(conn, host, config, true, from.Address, to, code)
}

func (s Sender) deliver(conn net.Conn, host string, config *tls.Config, startTLS bool, from, to, code string) error {
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer client.Close()
	if startTLS {
		if err = client.StartTLS(config); err != nil {
			return err
		}
	}
	if s.Username != "" {
		if s.Password == "" {
			return errors.New("SMTP password missing")
		}
		if err = client.Auth(smtp.PlainAuth("", s.Username, s.Password, host)); err != nil {
			return err
		}
	}
	if err = client.Mail(from); err != nil {
		return err
	}
	if err = client.Rcpt(to); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "From: %s\r\nTo: %s\r\nSubject: SAYA registration code\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\nКод подтверждения SAYA: %s\r\nДействует 10 минут.\r\n", from, to, code)
	if err != nil {
		return err
	}
	return w.Close()
}
