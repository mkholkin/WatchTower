package notification

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"net/textproto"
	"strings"
	"testing"
	"time"
)

func TestAuthSMTPDeliveryAndTLSRequirement(t *testing.T) {
	for _, allowInsecure := range []bool{false, true} {
		t.Run(map[bool]string{false: "requires TLS", true: "isolated SMTP"}[allowInsecure], func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			message := make(chan string, 1)
			go func() {
				conn, err := listener.Accept()
				if err != nil {
					message <- ""
					return
				}
				defer conn.Close()
				_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
				reader := textproto.NewReader(bufio.NewReader(conn))
				writer := textproto.NewWriter(bufio.NewWriter(conn))
				_ = writer.PrintfLine("220 localhost ESMTP")
				body := ""
				defer func() { message <- body }()
				for {
					line, err := reader.ReadLine()
					if err != nil {
						return
					}
					switch {
					case strings.HasPrefix(line, "EHLO"), strings.HasPrefix(line, "HELO"):
						_ = writer.PrintfLine("250 localhost")
					case line == "DATA":
						_ = writer.PrintfLine("354 send message")
						data, _ := reader.ReadDotBytes()
						body = string(data)
						_ = writer.PrintfLine("250 accepted")
					case line == "QUIT":
						_ = writer.PrintfLine("221 bye")
						return
					default:
						_ = writer.PrintfLine("250 OK")
					}
				}
			}()
			sender, err := NewAuthSMTPSender(listener.Addr().String(), "watchtower@example.test", allowInsecure)
			if err != nil {
				t.Fatal(err)
			}
			err = sender.Send(context.Background(), "alice@example.test", "login", "synthetic-code")
			body := <-message
			if !allowInsecure {
				if err == nil || body != "" {
					t.Fatal("SMTP without TLS must not send a secret")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			_, payload, found := strings.Cut(body, "\n\n")
			if !found {
				t.Fatal("missing mail body")
			}
			var decoded map[string]string
			if json.Unmarshal([]byte(payload), &decoded) != nil || decoded["kind"] != "login" || decoded["secret"] != "synthetic-code" {
				t.Fatal("invalid delivered payload")
			}
		})
	}
}
