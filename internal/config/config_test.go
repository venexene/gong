package config

import (
	"os"
	"testing"
)

func setEnv(t *testing.T, key, value string) {
	t.Helper()
	if err := os.Setenv(key, value); err != nil {
		t.Fatalf("failed to set %s: %v", key, err)
	}
}

func unsetEnv(t *testing.T, keys ...string) {
	t.Helper()
	for _, k := range keys {
		if err := os.Unsetenv(k); err != nil {
			t.Fatalf("failed to unset %s: %v", k, err)
		}
	}
}

func setAllRequired(t *testing.T) {
	t.Helper()
	setEnv(t, "DB_USER", "testuser")
	setEnv(t, "DB_PASSWORD", "testpass")
	setEnv(t, "DB_HOST", "localhost")
	setEnv(t, "DB_PORT", "5432")
	setEnv(t, "DB_NAME", "gong")
	setEnv(t, "RABBIT_USER", "guest")
	setEnv(t, "RABBIT_PASSWORD", "guest")
	setEnv(t, "RABBIT_HOST", "localhost")
	setEnv(t, "RABBIT_PORT", "5672")
}

func TestLoad_Success(t *testing.T) {
	unsetEnv(t,
		"HTTP_PORT", "DB_USER", "DB_PASSWORD", "DB_HOST", "DB_PORT", "DB_NAME",
		"RABBIT_USER", "RABBIT_PASSWORD", "RABBIT_HOST", "RABBIT_PORT",
	)
	setAllRequired(t)
	setEnv(t, "HTTP_PORT", "9090")
	defer unsetEnv(t,
		"HTTP_PORT", "DB_USER", "DB_PASSWORD", "DB_HOST", "DB_PORT", "DB_NAME",
		"RABBIT_USER", "RABBIT_PASSWORD", "RABBIT_HOST", "RABBIT_PORT",
	)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	if cfg.HTTPPort != "9090" {
		t.Errorf("expected HTTPPort '9090', got %q", cfg.HTTPPort)
	}

	expectedDSN := "postgres://testuser:testpass@localhost:5432/gong"
	if cfg.DB_DSN != expectedDSN {
		t.Errorf("expected DB_DSN %q, got %q", expectedDSN, cfg.DB_DSN)
	}

	expectedRabbitURL := "amqp://guest:guest@localhost:5672/"
	if cfg.RabbitURL != expectedRabbitURL {
		t.Errorf("expected RabbitURL %q, got %q", expectedRabbitURL, cfg.RabbitURL)
	}
}

func TestLoad_DefaultHTTPPort(t *testing.T) {
	unsetEnv(t,
		"HTTP_PORT", "DB_USER", "DB_PASSWORD", "DB_HOST", "DB_PORT", "DB_NAME",
		"RABBIT_USER", "RABBIT_PASSWORD", "RABBIT_HOST", "RABBIT_PORT",
	)
	setAllRequired(t)
	defer unsetEnv(t,
		"HTTP_PORT", "DB_USER", "DB_PASSWORD", "DB_HOST", "DB_PORT", "DB_NAME",
		"RABBIT_USER", "RABBIT_PASSWORD", "RABBIT_HOST", "RABBIT_PORT",
	)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	if cfg.HTTPPort != "8080" {
		t.Errorf("expected default HTTPPort '8080', got %q", cfg.HTTPPort)
	}
}

func TestLoad_MissingDBUser(t *testing.T) {
	unsetEnv(t,
		"HTTP_PORT", "DB_USER", "DB_PASSWORD", "DB_HOST", "DB_PORT", "DB_NAME",
		"RABBIT_USER", "RABBIT_PASSWORD", "RABBIT_HOST", "RABBIT_PORT",
	)
	setAllRequired(t)
	unsetEnv(t, "DB_USER")
	defer unsetEnv(t,
		"HTTP_PORT", "DB_USER", "DB_PASSWORD", "DB_HOST", "DB_PORT", "DB_NAME",
		"RABBIT_USER", "RABBIT_PASSWORD", "RABBIT_HOST", "RABBIT_PORT",
	)

	_, err := Load()
	if err == nil {
		t.Error("expected error for missing DB_USER, got nil")
	}
}

func TestLoad_MissingDBPassword(t *testing.T) {
	unsetEnv(t,
		"HTTP_PORT", "DB_USER", "DB_PASSWORD", "DB_HOST", "DB_PORT", "DB_NAME",
		"RABBIT_USER", "RABBIT_PASSWORD", "RABBIT_HOST", "RABBIT_PORT",
	)
	setAllRequired(t)
	unsetEnv(t, "DB_PASSWORD")
	defer unsetEnv(t,
		"HTTP_PORT", "DB_USER", "DB_PASSWORD", "DB_HOST", "DB_PORT", "DB_NAME",
		"RABBIT_USER", "RABBIT_PASSWORD", "RABBIT_HOST", "RABBIT_PORT",
	)

	_, err := Load()
	if err == nil {
		t.Error("expected error for missing DB_PASSWORD, got nil")
	}
}

func TestLoad_MissingDBHost(t *testing.T) {
	unsetEnv(t,
		"HTTP_PORT", "DB_USER", "DB_PASSWORD", "DB_HOST", "DB_PORT", "DB_NAME",
		"RABBIT_USER", "RABBIT_PASSWORD", "RABBIT_HOST", "RABBIT_PORT",
	)
	setAllRequired(t)
	unsetEnv(t, "DB_HOST")
	defer unsetEnv(t,
		"HTTP_PORT", "DB_USER", "DB_PASSWORD", "DB_HOST", "DB_PORT", "DB_NAME",
		"RABBIT_USER", "RABBIT_PASSWORD", "RABBIT_HOST", "RABBIT_PORT",
	)

	_, err := Load()
	if err == nil {
		t.Error("expected error for missing DB_HOST, got nil")
	}
}

func TestLoad_MissingDBPort(t *testing.T) {
	unsetEnv(t,
		"HTTP_PORT", "DB_USER", "DB_PASSWORD", "DB_HOST", "DB_PORT", "DB_NAME",
		"RABBIT_USER", "RABBIT_PASSWORD", "RABBIT_HOST", "RABBIT_PORT",
	)
	setAllRequired(t)
	unsetEnv(t, "DB_PORT")
	defer unsetEnv(t,
		"HTTP_PORT", "DB_USER", "DB_PASSWORD", "DB_HOST", "DB_PORT", "DB_NAME",
		"RABBIT_USER", "RABBIT_PASSWORD", "RABBIT_HOST", "RABBIT_PORT",
	)

	_, err := Load()
	if err == nil {
		t.Error("expected error for missing DB_PORT, got nil")
	}
}

func TestLoad_MissingDBName(t *testing.T) {
	unsetEnv(t,
		"HTTP_PORT", "DB_USER", "DB_PASSWORD", "DB_HOST", "DB_PORT", "DB_NAME",
		"RABBIT_USER", "RABBIT_PASSWORD", "RABBIT_HOST", "RABBIT_PORT",
	)
	setAllRequired(t)
	unsetEnv(t, "DB_NAME")
	defer unsetEnv(t,
		"HTTP_PORT", "DB_USER", "DB_PASSWORD", "DB_HOST", "DB_PORT", "DB_NAME",
		"RABBIT_USER", "RABBIT_PASSWORD", "RABBIT_HOST", "RABBIT_PORT",
	)

	_, err := Load()
	if err == nil {
		t.Error("expected error for missing DB_NAME, got nil")
	}
}

func TestLoad_MissingRabbitUser(t *testing.T) {
	unsetEnv(t,
		"HTTP_PORT", "DB_USER", "DB_PASSWORD", "DB_HOST", "DB_PORT", "DB_NAME",
		"RABBIT_USER", "RABBIT_PASSWORD", "RABBIT_HOST", "RABBIT_PORT",
	)
	setAllRequired(t)
	unsetEnv(t, "RABBIT_USER")
	defer unsetEnv(t,
		"HTTP_PORT", "DB_USER", "DB_PASSWORD", "DB_HOST", "DB_PORT", "DB_NAME",
		"RABBIT_USER", "RABBIT_PASSWORD", "RABBIT_HOST", "RABBIT_PORT",
	)

	_, err := Load()
	if err == nil {
		t.Error("expected error for missing RABBIT_USER, got nil")
	}
}

func TestLoad_MissingRabbitPassword(t *testing.T) {
	unsetEnv(t,
		"HTTP_PORT", "DB_USER", "DB_PASSWORD", "DB_HOST", "DB_PORT", "DB_NAME",
		"RABBIT_USER", "RABBIT_PASSWORD", "RABBIT_HOST", "RABBIT_PORT",
	)
	setAllRequired(t)
	unsetEnv(t, "RABBIT_PASSWORD")
	defer unsetEnv(t,
		"HTTP_PORT", "DB_USER", "DB_PASSWORD", "DB_HOST", "DB_PORT", "DB_NAME",
		"RABBIT_USER", "RABBIT_PASSWORD", "RABBIT_HOST", "RABBIT_PORT",
	)

	_, err := Load()
	if err == nil {
		t.Error("expected error for missing RABBIT_PASSWORD, got nil")
	}
}

func TestLoad_MissingRabbitHost(t *testing.T) {
	unsetEnv(t,
		"HTTP_PORT", "DB_USER", "DB_PASSWORD", "DB_HOST", "DB_PORT", "DB_NAME",
		"RABBIT_USER", "RABBIT_PASSWORD", "RABBIT_HOST", "RABBIT_PORT",
	)
	setAllRequired(t)
	unsetEnv(t, "RABBIT_HOST")
	defer unsetEnv(t,
		"HTTP_PORT", "DB_USER", "DB_PASSWORD", "DB_HOST", "DB_PORT", "DB_NAME",
		"RABBIT_USER", "RABBIT_PASSWORD", "RABBIT_HOST", "RABBIT_PORT",
	)

	_, err := Load()
	if err == nil {
		t.Error("expected error for missing RABBIT_HOST, got nil")
	}
}

func TestLoad_MissingRabbitPort(t *testing.T) {
	unsetEnv(t,
		"HTTP_PORT", "DB_USER", "DB_PASSWORD", "DB_HOST", "DB_PORT", "DB_NAME",
		"RABBIT_USER", "RABBIT_PASSWORD", "RABBIT_HOST", "RABBIT_PORT",
	)
	setAllRequired(t)
	unsetEnv(t, "RABBIT_PORT")
	defer unsetEnv(t,
		"HTTP_PORT", "DB_USER", "DB_PASSWORD", "DB_HOST", "DB_PORT", "DB_NAME",
		"RABBIT_USER", "RABBIT_PASSWORD", "RABBIT_HOST", "RABBIT_PORT",
	)

	_, err := Load()
	if err == nil {
		t.Error("expected error for missing RABBIT_PORT, got nil")
	}
}

func TestLoad_DSNFormat(t *testing.T) {
	unsetEnv(t,
		"HTTP_PORT", "DB_USER", "DB_PASSWORD", "DB_HOST", "DB_PORT", "DB_NAME",
		"RABBIT_USER", "RABBIT_PASSWORD", "RABBIT_HOST", "RABBIT_PORT",
	)
	setEnv(t, "DB_USER", "admin")
	setEnv(t, "DB_PASSWORD", "s3cr3t!")
	setEnv(t, "DB_HOST", "db.example.com")
	setEnv(t, "DB_PORT", "5433")
	setEnv(t, "DB_NAME", "gong_prod")
	setEnv(t, "RABBIT_USER", "rbuser")
	setEnv(t, "RABBIT_PASSWORD", "rbpass")
	setEnv(t, "RABBIT_HOST", "mq.example.com")
	setEnv(t, "RABBIT_PORT", "5673")
	defer unsetEnv(t,
		"HTTP_PORT", "DB_USER", "DB_PASSWORD", "DB_HOST", "DB_PORT", "DB_NAME",
		"RABBIT_USER", "RABBIT_PASSWORD", "RABBIT_HOST", "RABBIT_PORT",
	)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	expectedDSN := "postgres://admin:s3cr3t!@db.example.com:5433/gong_prod"
	if cfg.DB_DSN != expectedDSN {
		t.Errorf("expected DB_DSN %q, got %q", expectedDSN, cfg.DB_DSN)
	}

	expectedRabbitURL := "amqp://rbuser:rbpass@mq.example.com:5673/"
	if cfg.RabbitURL != expectedRabbitURL {
		t.Errorf("expected RabbitURL %q, got %q", expectedRabbitURL, cfg.RabbitURL)
	}
}
