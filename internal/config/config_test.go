package config

import "testing"

func TestLoadFromEnvDefaults(t *testing.T) {
	t.Setenv(envIMAPHost, "imap.example.com")
	t.Setenv(envIMAPUser, "user@example.com")
	t.Setenv(envIMAPPassword, "secret")
	t.Setenv(envIMAPPort, "")
	t.Setenv(envIMAPTLS, "")
	t.Setenv(envDefaultFolder, "")
	t.Setenv(envAuditLogPath, "")

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv() error = %v", err)
	}

	if cfg.Port != DefaultPort {
		t.Fatalf("Port = %d, want %d", cfg.Port, DefaultPort)
	}
	if !cfg.TLS {
		t.Fatalf("TLS = %v, want true", cfg.TLS)
	}
	if cfg.DefaultFolder != DefaultFolder {
		t.Fatalf("DefaultFolder = %q, want %q", cfg.DefaultFolder, DefaultFolder)
	}
	if cfg.DefaultPageSize != DefaultPageSize {
		t.Fatalf("DefaultPageSize = %d, want %d", cfg.DefaultPageSize, DefaultPageSize)
	}
	if cfg.AuditLogPath != DefaultAuditLog {
		t.Fatalf("AuditLogPath = %q, want %q", cfg.AuditLogPath, DefaultAuditLog)
	}
}

func TestLoadFromEnvMissingRequired(t *testing.T) {
	t.Setenv(envIMAPHost, "")
	t.Setenv(envIMAPUser, "")
	t.Setenv(envIMAPPassword, "")

	_, err := LoadFromEnv()
	if err == nil {
		t.Fatal("LoadFromEnv() error = nil, want error")
	}
}

func TestResolvePagination(t *testing.T) {
	cfg := Config{DefaultPageSize: 50}

	limit, offset, err := cfg.ResolvePagination(0, 10)
	if err != nil {
		t.Fatalf("ResolvePagination() error = %v", err)
	}
	if limit != 50 || offset != 10 {
		t.Fatalf("ResolvePagination() = (%d, %d), want (50, 10)", limit, offset)
	}

	if _, _, err := cfg.ResolvePagination(-1, 0); err == nil {
		t.Fatal("ResolvePagination() with negative limit error = nil, want error")
	}
	if _, _, err := cfg.ResolvePagination(1, -1); err == nil {
		t.Fatal("ResolvePagination() with negative offset error = nil, want error")
	}
}
