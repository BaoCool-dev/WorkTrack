// Package email gửi email qua SMTP (dev: Mailpit, B7).
package email

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/account"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

const defaultTimeout = 10 * time.Second

type Config struct {
	Host     string
	Port     int
	From     string // "WorkTrack <no-reply@worktrack.local>"
	Username string // để trống nếu server không yêu cầu xác thực
	Password string
	Timeout  time.Duration
}

// SMTPMailer cài đặt account.Mailer.
type SMTPMailer struct {
	cfg  Config
	from *mail.Address
	loc  *time.Location
}

func NewSMTPMailer(cfg Config) (*SMTPMailer, error) {
	from, err := mail.ParseAddress(cfg.From)
	if err != nil {
		return nil, fmt.Errorf("SMTP_FROM: %w", err)
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = defaultTimeout
	}
	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		loc = time.FixedZone("ICT", 7*3600)
	}
	return &SMTPMailer{cfg: cfg, from: from, loc: loc}, nil
}

func (m *SMTPMailer) SendActivation(ctx context.Context, e account.ActivationEmail) error {
	to, err := mail.ParseAddress(e.To)
	if err != nil {
		return fmt.Errorf("email.SendActivation: recipient: %w", err)
	}
	body := fmt.Sprintf(`Xin chào %s,

Quản trị viên đã tạo tài khoản WorkTrack cho bạn.
Mở liên kết dưới đây để đặt mật khẩu và kích hoạt tài khoản:

%s

Liên kết có hiệu lực đến %s và chỉ dùng được một lần.
Nếu bạn không yêu cầu tài khoản này, hãy bỏ qua email.

— WorkTrack
`, e.FullName, e.Link, e.ExpiresAt.In(m.loc).Format("15:04 02/01/2006"))
	msg, err := m.build(to, "Kích hoạt tài khoản WorkTrack", body)
	if err != nil {
		return err
	}
	return m.send(ctx, to.Address, msg)
}

func (m *SMTPMailer) build(to *mail.Address, subject, body string) ([]byte, error) {
	if strings.ContainsAny(subject, "\r\n") {
		return nil, errors.New("email: subject contains newline")
	}
	var b bytes.Buffer
	id := make([]byte, 12)
	_, _ = rand.Read(id)
	domainPart := m.from.Address[strings.LastIndex(m.from.Address, "@")+1:]
	headers := [][2]string{
		{"From", m.from.String()},
		{"To", to.String()},
		{"Subject", mime.QEncoding.Encode("utf-8", subject)},
		{"Date", time.Now().Format(time.RFC1123Z)},
		{"Message-ID", "<" + hex.EncodeToString(id) + "@" + domainPart + ">"},
		{"MIME-Version", "1.0"},
		{"Content-Type", "text/plain; charset=utf-8"},
		{"Content-Transfer-Encoding", "quoted-printable"},
	}
	for _, h := range headers {
		if strings.ContainsAny(h[1], "\r\n") {
			return nil, fmt.Errorf("email: header %s contains newline", h[0])
		}
		fmt.Fprintf(&b, "%s: %s\r\n", h[0], h[1])
	}
	b.WriteString("\r\n")
	qp := quotedprintable.NewWriter(&b)
	if _, err := qp.Write([]byte(strings.ReplaceAll(body, "\n", "\r\n"))); err != nil {
		return nil, fmt.Errorf("email: encode body: %w", err)
	}
	if err := qp.Close(); err != nil {
		return nil, fmt.Errorf("email: encode body: %w", err)
	}
	return b.Bytes(), nil
}

// send gửi một thư với deadline; lỗi được bọc bằng domain.ErrEmailDeliveryFailed.
func (m *SMTPMailer) send(ctx context.Context, to string, msg []byte) error {
	if err := m.sendRaw(ctx, to, msg); err != nil {
		return fmt.Errorf("%w: %w", domain.ErrEmailDeliveryFailed, err)
	}
	return nil
}

func (m *SMTPMailer) sendRaw(ctx context.Context, to string, msg []byte) error {
	ctx, cancel := context.WithTimeout(ctx, m.cfg.Timeout)
	defer cancel()
	addr := net.JoinHostPort(m.cfg.Host, strconv.Itoa(m.cfg.Port))
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	deadline, _ := ctx.Deadline()
	_ = conn.SetDeadline(deadline)
	c, err := smtp.NewClient(conn, m.cfg.Host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("smtp client: %w", err)
	}
	defer func() { _ = c.Close() }()
	if ok, _ := c.Extension("STARTTLS"); ok {
		if err := c.StartTLS(&tls.Config{ServerName: m.cfg.Host, MinVersion: tls.VersionTLS12}); err != nil {
			return fmt.Errorf("starttls: %w", err)
		}
	}
	if m.cfg.Username != "" {
		// PlainAuth của net/smtp từ chối gửi mật khẩu qua kết nối không TLS (trừ localhost).
		if err := c.Auth(smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)); err != nil {
			return fmt.Errorf("auth: %w", err)
		}
	}
	if err := c.Mail(m.from.Address); err != nil {
		return fmt.Errorf("mail from: %w", err)
	}
	if err := c.Rcpt(to); err != nil {
		return fmt.Errorf("rcpt: %w", err)
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("data: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("close data: %w", err)
	}
	return c.Quit()
}
