package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

const (
	DefaultFolder    = "INBOX"
	DefaultPort      = 993
	DefaultPageSize  = 50
	DefaultAuditLog  = "imap-mcp-audit.log"
	envIMAPHost      = "IMAP_HOST"
	envIMAPPort      = "IMAP_PORT"
	envIMAPUser      = "IMAP_USER"
	envIMAPPassword  = "IMAP_PASSWORD"
	envIMAPTLS       = "IMAP_TLS"
	envDefaultFolder = "IMAP_DEFAULT_FOLDER"
	envAuditLogPath  = "IMAP_AUDIT_LOG"
)

type Config struct {
	Host            string
	Port            int
	User            string
	Password        string
	TLS             bool
	DefaultFolder   string
	DefaultPageSize int
	AuditLogPath    string
}

func LoadFromEnv() (Config, error) {
	cfg := Config{
		Host:            strings.TrimSpace(os.Getenv(envIMAPHost)),
		User:            strings.TrimSpace(os.Getenv(envIMAPUser)),
		Password:        os.Getenv(envIMAPPassword),
		TLS:             true,
		DefaultFolder:   DefaultFolder,
		DefaultPageSize: DefaultPageSize,
		AuditLogPath:    DefaultAuditLog,
	}

	portValue := strings.TrimSpace(os.Getenv(envIMAPPort))
	if portValue == "" {
		cfg.Port = DefaultPort
	} else {
		port, err := strconv.Atoi(portValue)
		if err != nil {
			return Config{}, fmt.Errorf("parse %s: %w", envIMAPPort, err)
		}
		cfg.Port = port
	}

	tlsValue := strings.TrimSpace(os.Getenv(envIMAPTLS))
	if tlsValue != "" {
		tlsEnabled, err := strconv.ParseBool(tlsValue)
		if err != nil {
			return Config{}, fmt.Errorf("parse %s: %w", envIMAPTLS, err)
		}
		cfg.TLS = tlsEnabled
	}

	if folder := strings.TrimSpace(os.Getenv(envDefaultFolder)); folder != "" {
		cfg.DefaultFolder = folder
	}

	if auditLogPath := strings.TrimSpace(os.Getenv(envAuditLogPath)); auditLogPath != "" {
		cfg.AuditLogPath = auditLogPath
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) Validate() error {
	var errs []error

	if c.Host == "" {
		errs = append(errs, fmt.Errorf("%s is required", envIMAPHost))
	}
	if c.Port <= 0 || c.Port > 65535 {
		errs = append(errs, fmt.Errorf("%s must be between 1 and 65535", envIMAPPort))
	}
	if c.User == "" {
		errs = append(errs, fmt.Errorf("%s is required", envIMAPUser))
	}
	if c.Password == "" {
		errs = append(errs, fmt.Errorf("%s is required", envIMAPPassword))
	}
	if strings.TrimSpace(c.DefaultFolder) == "" {
		errs = append(errs, errors.New("default folder cannot be blank"))
	}
	if c.DefaultPageSize <= 0 {
		errs = append(errs, errors.New("default page size must be greater than zero"))
	}
	if strings.TrimSpace(c.AuditLogPath) == "" {
		errs = append(errs, errors.New("audit log path cannot be blank"))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

func (c Config) Address() string {
	return net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
}

func (c Config) ResolveFolder(folder string) string {
	if strings.TrimSpace(folder) == "" {
		return c.DefaultFolder
	}
	return folder
}

func (c Config) ResolvePagination(limit, offset int) (int, int, error) {
	if limit < 0 {
		return 0, 0, errors.New("limit cannot be negative")
	}
	if offset < 0 {
		return 0, 0, errors.New("offset cannot be negative")
	}
	if limit == 0 {
		limit = c.DefaultPageSize
	}
	return limit, offset, nil
}
