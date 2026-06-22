package api

import (
	"encoding/json"
	"testing"

	"github.com/worryzyy/upstream-hub/internal/storage"
)

func TestPrepareEmailNotifyConfigKeepsExistingPassword(t *testing.T) {
	oldRaw := `{"host":"smtp.old.example","port":465,"username":"old@example.com","password":"old-secret","from":"old@example.com","from_name":"Old","to":["ops@example.com"],"use_tls":true}`
	nextRaw := `{"from_name":"Sub2API","password":""}`

	out, err := prepareNotifyConfig(storage.NotifyEmail, nextRaw, oldRaw)
	if err != nil {
		t.Fatal(err)
	}

	var cfg map[string]any
	if err := json.Unmarshal([]byte(out), &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg["password"] != "old-secret" {
		t.Fatalf("password = %v, want old-secret", cfg["password"])
	}
	if cfg["host"] != "smtp.old.example" {
		t.Fatalf("host = %v, want existing host", cfg["host"])
	}
	if cfg["from_name"] != "Sub2API" {
		t.Fatalf("from_name = %v, want Sub2API", cfg["from_name"])
	}
}

func TestPrepareEmailNotifyConfigUsesEnvironmentDefaults(t *testing.T) {
	t.Setenv("UPSTREAM_HUB_SMTP_USERNAME", "mailer@example.com")
	t.Setenv("UPSTREAM_HUB_SMTP_PASSWORD", "env-secret")
	t.Setenv("UPSTREAM_HUB_SMTP_TO", "ops@example.com")

	out, err := prepareNotifyConfig(storage.NotifyEmail, `{}`, "")
	if err != nil {
		t.Fatal(err)
	}

	var cfg struct {
		Host     string   `json:"host"`
		Port     int      `json:"port"`
		Username string   `json:"username"`
		Password string   `json:"password"`
		From     string   `json:"from"`
		FromName string   `json:"from_name"`
		To       []string `json:"to"`
		UseTLS   bool     `json:"use_tls"`
	}
	if err := json.Unmarshal([]byte(out), &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Host != "smtp.gmail.com" || cfg.Port != 465 || !cfg.UseTLS {
		t.Fatalf("smtp defaults = %s:%d tls=%v", cfg.Host, cfg.Port, cfg.UseTLS)
	}
	if cfg.Username != "mailer@example.com" || cfg.From != "mailer@example.com" {
		t.Fatalf("sender defaults = username %q from %q", cfg.Username, cfg.From)
	}
	if cfg.Password != "env-secret" {
		t.Fatalf("password = %q, want env-secret", cfg.Password)
	}
	if len(cfg.To) != 1 || cfg.To[0] != "ops@example.com" {
		t.Fatalf("to = %#v, want ops@example.com", cfg.To)
	}
}
