package imapclient

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chrisw-dev/imap-mcp-go/internal/config"
)

func TestExtractMessageBodyPrefersPlainText(t *testing.T) {
	raw := strings.Join([]string{
		"From: sender@example.com",
		"To: you@example.com",
		"Subject: Multipart",
		"MIME-Version: 1.0",
		"Content-Type: multipart/alternative; boundary=frontier",
		"",
		"--frontier",
		"Content-Type: text/plain; charset=utf-8",
		"",
		"Plain text wins.",
		"--frontier",
		"Content-Type: text/html; charset=utf-8",
		"",
		"<html><body><p>HTML fallback</p></body></html>",
		"--frontier--",
		"",
	}, "\r\n")

	body, contentType, err := extractMessageBody([]byte(raw))
	if err != nil {
		t.Fatalf("extractMessageBody() error = %v", err)
	}
	if body != "Plain text wins." {
		t.Fatalf("body = %q, want %q", body, "Plain text wins.")
	}
	if contentType != "text/plain" {
		t.Fatalf("contentType = %q, want %q", contentType, "text/plain")
	}
}

func TestExtractMessageBodyFallsBackToHTML(t *testing.T) {
	raw := strings.Join([]string{
		"From: sender@example.com",
		"To: you@example.com",
		"Subject: HTML only",
		"MIME-Version: 1.0",
		"Content-Type: text/html; charset=utf-8",
		"",
		"<html><head><title>Ignore</title><script>alert('x')</script></head><body><h1>Hello</h1><p>World</p></body></html>",
		"",
	}, "\r\n")

	body, contentType, err := extractMessageBody([]byte(raw))
	if err != nil {
		t.Fatalf("extractMessageBody() error = %v", err)
	}
	if body != "Hello World" {
		t.Fatalf("body = %q, want %q", body, "Hello World")
	}
	if contentType != "text/html" {
		t.Fatalf("contentType = %q, want %q", contentType, "text/html")
	}
}

func TestFrameUntrustedContent(t *testing.T) {
	framed := frameUntrustedContent("Hello")

	if !strings.Contains(framed, untrustedBegin) {
		t.Fatalf("framed body missing begin marker: %q", framed)
	}
	if !strings.Contains(framed, untrustedEnd) {
		t.Fatalf("framed body missing end marker: %q", framed)
	}
}

func TestAppendAuditEntry(t *testing.T) {
	client := &Client{
		cfg: config.Config{
			AuditLogPath: filepath.Join(t.TempDir(), "audit.log"),
		},
	}

	err := client.appendAuditEntry(auditEntry{
		Action: "mark_read",
		UID:    42,
		Folder: "INBOX",
	})
	if err != nil {
		t.Fatalf("appendAuditEntry() error = %v", err)
	}

	data, err := os.ReadFile(client.cfg.AuditLogPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(data), "\"action\":\"mark_read\"") {
		t.Fatalf("audit log = %q, want action entry", string(data))
	}
}
