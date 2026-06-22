package notify

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"mime"
	"net"
	"net/smtp"
	"strings"
	"time"

	"github.com/worryzyy/upstream-hub/internal/storage"
)

func init() {
	Register(storage.NotifyEmail, func(raw string) (Notifier, error) { return newEmail(raw) })
}

type emailConfig struct {
	Host     string   `json:"host"`      // smtp.example.com
	Port     int      `json:"port"`      // 465 / 587
	Username string   `json:"username"`  // SMTP 用户名
	Password string   `json:"password"`  // SMTP 密码 / 授权码
	From     string   `json:"from"`      // 发件人邮箱（可与 Username 不同）
	FromName string   `json:"from_name"` // 发件人名称
	To       []string `json:"to"`        // 收件人列表
	UseTLS   bool     `json:"use_tls"`   // 是否使用隐式 TLS（一般 465 端口）
}

type email struct{ cfg emailConfig }

func newEmail(raw string) (*email, error) {
	var cfg emailConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return nil, err
	}
	if cfg.Host == "" || cfg.Port == 0 || cfg.From == "" || len(cfg.To) == 0 {
		return nil, errors.New("email config requires host/port/from/to")
	}
	return &email{cfg: cfg}, nil
}

func (e *email) Type() storage.NotificationChannelType { return storage.NotifyEmail }

func (e *email) Send(ctx context.Context, msg Message) error {
	addr := fmt.Sprintf("%s:%d", e.cfg.Host, e.cfg.Port)
	auth := smtpAuth(e.cfg)

	body := buildEmailBody(e.cfg.FromName, e.cfg.From, e.cfg.To, msg.Subject, msg.Body)

	// 简单 deadline，避免完全阻塞调度。
	done := make(chan error, 1)
	go func() {
		if e.cfg.UseTLS {
			done <- sendTLS(addr, e.cfg.Host, auth, e.cfg.From, e.cfg.To, []byte(body))
			return
		}
		done <- sendPlain(addr, e.cfg.Host, auth, e.cfg.From, e.cfg.To, []byte(body))
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(45 * time.Second):
		return errors.New("smtp send timeout")
	}
}

func smtpAuth(cfg emailConfig) smtp.Auth {
	if cfg.Username == "" && cfg.Password == "" {
		return nil
	}
	return smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
}

func buildEmailBody(fromName, from string, to []string, subject, body string) string {
	fromHeader := sanitizeEmailHeader(from)
	if name := sanitizeEmailHeader(fromName); name != "" {
		fromHeader = fmt.Sprintf("%s <%s>", mime.QEncoding.Encode("UTF-8", name), fromHeader)
	}
	headers := []string{
		"From: " + fromHeader,
		"To: " + sanitizeEmailHeader(strings.Join(to, ", ")),
		"Subject: " + mime.QEncoding.Encode("UTF-8", sanitizeEmailHeader(subject)),
		"Date: " + time.Now().Format(time.RFC1123Z),
		"MIME-Version: 1.0",
		"Content-Type: text/html; charset=UTF-8",
		"Content-Transfer-Encoding: 8bit",
	}
	return strings.Join(headers, "\r\n") + "\r\n\r\n" + buildHTMLBody(subject, body)
}

func sanitizeEmailHeader(s string) string {
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.TrimSpace(s)
}

func buildHTMLBody(subject, body string) string {
	escapedSubject := html.EscapeString(subject)
	escapedBody := html.EscapeString(body)
	escapedBody = strings.ReplaceAll(escapedBody, "\n", "<br>")
	return fmt.Sprintf(`<!doctype html>
<html>
<body style="margin:0;background:#f6f7f9;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;color:#172033;">
  <div style="max-width:640px;margin:0 auto;padding:24px;">
    <div style="background:#ffffff;border:1px solid #e5e7eb;border-radius:8px;padding:20px;">
      <h2 style="margin:0 0 12px;font-size:18px;line-height:1.4;color:#111827;">%s</h2>
      <div style="font-size:14px;line-height:1.7;color:#374151;">%s</div>
    </div>
    <p style="margin:12px 0 0;font-size:12px;color:#6b7280;">upstream-hub</p>
  </div>
</body>
</html>`, escapedSubject, escapedBody)
}

const smtpDialTimeout = 10 * time.Second
const smtpIOTimeout = 20 * time.Second

func sendPlain(addr, host string, auth smtp.Auth, from string, to []string, body []byte) error {
	dialer := &net.Dialer{Timeout: smtpDialTimeout}
	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}
	_ = conn.SetDeadline(time.Now().Add(smtpIOTimeout))
	defer conn.Close()

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("smtp new client: %w", err)
	}
	defer client.Close()

	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}); err != nil {
			return fmt.Errorf("smtp starttls: %w", err)
		}
	}
	return sendWithClient(client, auth, from, to, body)
}

// sendTLS 通过 SMTPS（隐式 TLS，常见于 465）发送邮件。
func sendTLS(addr, host string, auth smtp.Auth, from string, to []string, body []byte) error {
	dialer := &net.Dialer{Timeout: smtpDialTimeout}
	tlsConfig := &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}
	conn, err := tls.DialWithDialer(dialer, "tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("smtp tls dial: %w", err)
	}
	_ = conn.SetDeadline(time.Now().Add(smtpIOTimeout))
	defer conn.Close()

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("smtp new client: %w", err)
	}
	defer client.Close()

	return sendWithClient(client, auth, from, to, body)
}

func sendWithClient(client *smtp.Client, auth smtp.Auth, from string, to []string, body []byte) error {
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("smtp mail: %w", err)
	}
	for _, rcpt := range to {
		if err := client.Rcpt(rcpt); err != nil {
			return fmt.Errorf("smtp rcpt %s: %w", rcpt, err)
		}
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := w.Write(body); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp close: %w", err)
	}
	_ = client.Quit()
	return nil
}
