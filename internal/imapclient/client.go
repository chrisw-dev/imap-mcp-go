package imapclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"os"
	"strings"
	"time"

	"github.com/chrisw-dev/imap-mcp-go/internal/config"
	"github.com/emersion/go-imap/v2"
	imapcli "github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-message/charset"
)

var ErrMessageNotFound = errors.New("message not found")

type Client struct {
	cfg         config.Config
	dialOptions *imapcli.Options
}

type MessageSummary struct {
	UID     uint32    `json:"uid"`
	Folder  string    `json:"folder"`
	Sender  string    `json:"sender"`
	Subject string    `json:"subject"`
	Date    time.Time `json:"date"`
	Size    int64     `json:"size"`
}

type MessagePage struct {
	Folder   string           `json:"folder"`
	Limit    int              `json:"limit"`
	Offset   int              `json:"offset"`
	Total    int              `json:"total"`
	Messages []MessageSummary `json:"messages"`
}

type MessageBody struct {
	UID         uint32    `json:"uid"`
	Folder      string    `json:"folder"`
	Sender      string    `json:"sender"`
	Subject     string    `json:"subject"`
	Date        time.Time `json:"date"`
	ContentType string    `json:"content_type"`
	Body        string    `json:"body"`
	FramedBody  string    `json:"framed_body"`
}

type MutationResult struct {
	Status       string `json:"status"`
	Action       string `json:"action"`
	UID          uint32 `json:"uid"`
	Folder       string `json:"folder"`
	TargetFolder string `json:"target_folder,omitempty"`
}

type auditEntry struct {
	Timestamp    time.Time `json:"timestamp"`
	Action       string    `json:"action"`
	UID          uint32    `json:"uid"`
	Folder       string    `json:"folder"`
	TargetFolder string    `json:"target_folder,omitempty"`
}

func New(cfg config.Config) *Client {
	return &Client{
		cfg: cfg,
		dialOptions: &imapcli.Options{
			WordDecoder: &mime.WordDecoder{CharsetReader: charset.Reader},
		},
	}
}

func withMailbox[T any](ctx context.Context, c *Client, folder string, fn func(*imapcli.Client, string) (T, error)) (T, error) {
	var zero T

	if err := ctx.Err(); err != nil {
		return zero, err
	}

	session, err := c.dial()
	if err != nil {
		return zero, err
	}
	defer session.Close()

	if err := session.Login(c.cfg.User, c.cfg.Password).Wait(); err != nil {
		return zero, fmt.Errorf("login to IMAP server: %w", err)
	}

	loggedOut := false
	defer func() {
		if !loggedOut {
			_ = session.Logout().Wait()
		}
	}()

	resolvedFolder := c.cfg.ResolveFolder(folder)
	if _, err := session.Select(resolvedFolder, nil).Wait(); err != nil {
		return zero, fmt.Errorf("select folder %q: %w", resolvedFolder, err)
	}

	value, err := fn(session, resolvedFolder)
	if err != nil {
		return zero, err
	}

	if err := session.Logout().Wait(); err != nil {
		return zero, fmt.Errorf("logout from IMAP server: %w", err)
	}
	loggedOut = true

	return value, nil
}

func (c *Client) dial() (*imapcli.Client, error) {
	if c.cfg.TLS {
		session, err := imapcli.DialTLS(c.cfg.Address(), c.dialOptions)
		if err != nil {
			return nil, fmt.Errorf("dial TLS IMAP server %q: %w", c.cfg.Address(), err)
		}
		return session, nil
	}

	session, err := imapcli.DialInsecure(c.cfg.Address(), c.dialOptions)
	if err != nil {
		return nil, fmt.Errorf("dial IMAP server %q: %w", c.cfg.Address(), err)
	}
	return session, nil
}

func paginateUIDs(uids []imap.UID, limit, offset int) []imap.UID {
	if offset >= len(uids) {
		return nil
	}

	end := offset + limit
	if end > len(uids) {
		end = len(uids)
	}

	return uids[offset:end]
}

func reverseUIDs(uids []imap.UID) {
	for left, right := 0, len(uids)-1; left < right; left, right = left+1, right-1 {
		uids[left], uids[right] = uids[right], uids[left]
	}
}

func formatEnvelopeAddress(addresses []imap.Address) string {
	if len(addresses) == 0 {
		return ""
	}

	addr := addresses[0]
	email := addr.Addr()

	switch {
	case addr.Name != "" && email != "":
		return fmt.Sprintf("%s <%s>", addr.Name, email)
	case email != "":
		return email
	default:
		return addr.Name
	}
}

func formatSender(envelope *imap.Envelope) string {
	if envelope == nil {
		return ""
	}

	if sender := formatEnvelopeAddress(envelope.From); sender != "" {
		return sender
	}

	return formatEnvelopeAddress(envelope.Sender)
}

func envelopeDate(envelope *imap.Envelope) time.Time {
	if envelope == nil {
		return time.Time{}
	}
	return envelope.Date
}

func envelopeSubject(envelope *imap.Envelope) string {
	if envelope == nil {
		return ""
	}
	return envelope.Subject
}

func (c *Client) appendAuditEntry(entry auditEntry) error {
	file, err := os.OpenFile(c.cfg.AuditLogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open audit log: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(entry); err != nil {
		return fmt.Errorf("write audit log: %w", err)
	}

	return nil
}

func normalizeBodyText(body string) string {
	return strings.TrimSpace(strings.ReplaceAll(body, "\u00a0", " "))
}
