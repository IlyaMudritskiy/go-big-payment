package config

import "testing"

func TestLoad(t *testing.T) {
	t.Run("missing DATABASE_URL", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "")
		if _, err := Load(); err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("defaults applied", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "postgres://x")
		cfg, err := Load()

		if err != nil {
			t.Fatal("unexpected error: %w", err)
		}

		if cfg.HttpAddr != ":8080" {
			t.Errorf("HTTP_ADDR = %q, want %q", cfg.HttpAddr, ":8080")
		}
	})
}
