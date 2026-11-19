package email

import (
	"bufio"
	"context"
	"errors"
	"io"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"strings"
	"testing"
	"time"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/account"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

// fakeSMTP nhận một thư rồi gửi nội dung DATA qua kênh.
func fakeSMTP(t *testing.T) (port int, got <-chan string) {
	t.Helper()
	ln, err := new(net.ListenConfig).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	ch := make(chan string, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		r := bufio.NewReader(conn)
		w := func(s string) { _, _ = io.WriteString(conn, s+"\r\n") }
		w("220 fake ESMTP")
		var data strings.Builder
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				return
			}
			cmd := strings.ToUpper(strings.TrimSpace(line))
			switch {
			case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
				w("250 fake")
			case strings.HasPrefix(cmd, "DATA"):
				w("354 go")
				for {
					l, err := r.ReadString('\n')
					if err != nil {
						return
					}
					if l == ".\r\n" {
						break
					}
					data.WriteString(l)
				}
				w("250 ok")
				ch <- data.String()
			case strings.HasPrefix(cmd, "QUIT"):
				w("221 bye")
				return
			default:
				w("250 ok")
			}
		}
	}()
	return ln.Addr().(*net.TCPAddr).Port, ch
}

func TestSMTPMailer_SendActivation_CF01_B7(t *testing.T) {
	port, got := fakeSMTP(t)
	m, err := NewSMTPMailer(Config{Host: "127.0.0.1", Port: port, From: "WorkTrack <no-reply@worktrack.local>"})
	if err != nil {
		t.Fatal(err)
	}
	exp := time.Date(2026, 11, 22, 3, 0, 0, 0, time.UTC)
	link := "http://localhost:5173/activate?token=abc"
	err = m.SendActivation(t.Context(), account.ActivationEmail{
		To: "an@worktrack.local", FullName: "Nguyễn Văn An", Link: link, ExpiresAt: exp,
	})
	if err != nil {
		t.Fatal(err)
	}
	raw := <-got
	msg, err := mail.ReadMessage(strings.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	subj, _ := new(mime.WordDecoder).DecodeHeader(msg.Header.Get("Subject"))
	if subj != "Kích hoạt tài khoản WorkTrack" || msg.Header.Get("To") != "<an@worktrack.local>" {
		t.Fatalf("header: %q %q", subj, msg.Header.Get("To"))
	}
	body, _ := io.ReadAll(quotedprintable.NewReader(msg.Body))
	for _, want := range []string{"Nguyễn Văn An", link, "10:00 22/11/2026"} {
		if !strings.Contains(string(body), want) {
			t.Errorf("thân thư thiếu %q:\n%s", want, body)
		}
	}
}

func TestSMTPMailer_Unreachable_ReturnsDeliveryFailed_CF01_B7(t *testing.T) {
	ln, _ := new(net.ListenConfig).Listen(t.Context(), "tcp", "127.0.0.1:0")
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()
	m, _ := NewSMTPMailer(Config{Host: "127.0.0.1", Port: port, From: "no-reply@worktrack.local", Timeout: time.Second})
	err := m.SendActivation(context.Background(), account.ActivationEmail{To: "a@b.vn", FullName: "A", Link: "x"})
	if !errors.Is(err, domain.ErrEmailDeliveryFailed) {
		t.Fatalf("got %v", err)
	}
}

func TestSMTPMailer_InvalidAddresses(t *testing.T) {
	if _, err := NewSMTPMailer(Config{From: "khong hop le"}); err == nil {
		t.Fatal("from sai phải lỗi")
	}
	m, _ := NewSMTPMailer(Config{From: "a@b.vn"})
	if err := m.SendActivation(t.Context(), account.ActivationEmail{To: "x\r\nBcc: y@z.vn"}); err == nil {
		t.Fatal("người nhận chứa xuống dòng phải lỗi")
	}
	if _, err := m.build(&mail.Address{Address: "a@b.vn"}, "a\r\nBcc: x", "b"); err == nil {
		t.Fatal("header injection phải lỗi")
	}
}
