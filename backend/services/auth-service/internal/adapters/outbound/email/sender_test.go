package email

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"
	"time"
)

func TestDeliverViaSTARTTLS(t *testing.T) {
	// Use a test-only trust root without modifying the host certificate store.
	authority := httptest.NewTLSServer(nil)
	defer authority.Close()
	roots := x509.NewCertPool()
	roots.AddCert(authority.Certificate())
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()
	deadline := time.Now().Add(5 * time.Second)
	server.SetDeadline(deadline)
	client.SetDeadline(deadline)
	received := make(chan string, 1)
	go func() {
		conn := textproto.NewConn(server)
		defer conn.Close()
		if conn.PrintfLine("220 test SMTP") != nil {
			return
		}
		if _, e := conn.ReadLine(); e != nil {
			return
		}
		if conn.PrintfLine("250-test\r\n250 STARTTLS") != nil {
			return
		}
		line, e := conn.ReadLine()
		if e != nil || line != "STARTTLS" {
			return
		}
		if conn.PrintfLine("220 Ready") != nil {
			return
		}
		secure := tls.Server(server, authority.TLS)
		defer server.Close()
		reader := textproto.NewReader(bufio.NewReader(secure))
		for {
			line, e := reader.ReadLine()
			if e != nil {
				return
			}
			switch {
			case strings.HasPrefix(line, "EHLO "), strings.HasPrefix(line, "MAIL FROM:"), strings.HasPrefix(line, "RCPT TO:"):
				fmt.Fprint(secure, "250 OK\r\n")
			case line == "DATA":
				fmt.Fprint(secure, "354 Send message\r\n")
				body, e := io.ReadAll(reader.DotReader())
				if e != nil {
					return
				}
				received <- string(body)
				fmt.Fprint(secure, "250 Accepted\r\n")
				return
			default:
				return
			}
		}
	}()
	sender := Sender{}
	config := &tls.Config{RootCAs: roots, ServerName: "example.com", MinVersion: tls.VersionTLS12}
	if err := sender.deliver(client, "example.com", config, true, "saya@example.test", "person@example.test", "012345"); err != nil {
		t.Fatal(err)
	}
	if body := <-received; !strings.Contains(body, "012345") || !strings.Contains(body, "To: person@example.test") {
		t.Fatal("recipient or code missing")
	}
}
