package notification

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"
)

// AuthSMTPSender requires STARTTLS unless explicitly configured for an isolated relay.
type AuthSMTPSender struct {
	address, host, from string
	allowInsecure       bool
}

func NewAuthSMTPSender(address, from string, allowInsecure bool) (*AuthSMTPSender, error) {
	host, _, err := net.SplitHostPort(address)
	if err != nil || host == "" || !plainEmail(from) {
		return nil, errors.New("invalid authentication SMTP configuration")
	}
	return &AuthSMTPSender{address: address, host: host, from: from, allowInsecure: allowInsecure}, nil
}

func plainEmail(value string) bool {
	address, err := mail.ParseAddress(value)
	return err == nil && address.Address == value && !strings.ContainsAny(value, "\r\n")
}

func (s *AuthSMTPSender) Send(ctx context.Context, to, kind, secret string) error {
	if !plainEmail(to) {
		return errors.New("invalid authentication recipient")
	}
	conn, err := (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, "tcp", s.address)
	if err != nil {
		return errors.New("authentication SMTP unavailable")
	}
	defer conn.Close()
	deadline := time.Now().Add(10 * time.Second)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		deadline = ctxDeadline
	}
	if err = conn.SetDeadline(deadline); err != nil {
		return err
	}
	client, err := smtp.NewClient(conn, s.host)
	if err != nil {
		return errors.New("authentication SMTP handshake failed")
	}
	defer client.Close()
	if supported, _ := client.Extension("STARTTLS"); supported {
		if err = client.StartTLS(&tls.Config{ServerName: s.host, MinVersion: tls.VersionTLS12}); err != nil {
			return errors.New("authentication SMTP TLS failed")
		}
	} else if !s.allowInsecure {
		return errors.New("authentication SMTP requires TLS")
	}
	if err = client.Mail(s.from); err != nil {
		return errors.New("authentication SMTP sender rejected")
	}
	if err = client.Rcpt(to); err != nil {
		return errors.New("authentication SMTP recipient rejected")
	}
	writer, err := client.Data()
	if err != nil {
		return errors.New("authentication SMTP delivery failed")
	}
	payload, _ := json.Marshal(map[string]string{"kind": kind, "secret": secret})
	message := "From: " + s.from + "\r\nTo: " + to + "\r\nSubject: WatchTower verification\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n" + string(payload) + "\r\n"
	if _, err = writer.Write([]byte(message)); err != nil {
		return errors.New("authentication SMTP delivery failed")
	}
	if err = writer.Close(); err != nil {
		return errors.New("authentication SMTP delivery rejected")
	}
	// DATA acceptance is authoritative; a disconnected QUIT must not roll back a delivered code.
	_ = client.Quit()
	return nil
}
