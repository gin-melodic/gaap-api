package demo_data

import (
	"testing"
	"time"
)

func TestLoadConfigDefaultsToDisabled(t *testing.T) {
	t.Setenv("DEMO_DATA_GENERATOR_ENABLED", "")
	t.Setenv("ONLINE_DEMO_USER_EMAIL", "")
	t.Setenv("ONLINE_DEMO_USER_PASSWORD", "")
	t.Setenv("DEMO_DATA_START_DATE", "")
	t.Setenv("DEMO_DATA_TIMEZONE", "")

	config, err := LoadConfig(t.Context())
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	if config.Enabled {
		t.Fatal("generator should be disabled by default")
	}
	if config.OnlineDemoEnabled {
		t.Fatal("online demo should be disabled by default")
	}
	if config.StartDate.Format(time.DateOnly) != defaultStartDate {
		t.Fatalf("start date = %s, want %s", config.StartDate.Format(time.DateOnly), defaultStartDate)
	}
	if config.InitialBackfillEndDate.Format(time.DateOnly) != defaultInitialBackfillEndDate {
		t.Fatalf("initial end date = %s, want %s", config.InitialBackfillEndDate.Format(time.DateOnly), defaultInitialBackfillEndDate)
	}
	if config.Location.String() != defaultTimezone {
		t.Fatalf("timezone = %s, want %s", config.Location, defaultTimezone)
	}
}

func TestLoadConfigRequiresEmailWhenEnabled(t *testing.T) {
	t.Setenv("DEMO_DATA_GENERATOR_ENABLED", "true")
	t.Setenv("ONLINE_DEMO_USER_EMAIL", "")
	t.Setenv("ONLINE_DEMO_USER_PASSWORD", "")
	t.Setenv("DEMO_DATA_START_DATE", "")
	t.Setenv("DEMO_DATA_TIMEZONE", "")

	if _, err := LoadConfig(t.Context()); err == nil {
		t.Fatal("expected enabled generator without email to fail")
	}
}

func TestLoadConfigNormalizesEmailAndCustomDate(t *testing.T) {
	t.Setenv("DEMO_DATA_GENERATOR_ENABLED", "TRUE")
	t.Setenv("ONLINE_DEMO_USER_EMAIL", " Demo@Example.COM ")
	t.Setenv("ONLINE_DEMO_USER_PASSWORD", "demo-password")
	t.Setenv("DEMO_DATA_START_DATE", "2026-05-06")
	t.Setenv("DEMO_DATA_TIMEZONE", "UTC")

	config, err := LoadConfig(t.Context())
	if err != nil {
		t.Fatalf("load custom config: %v", err)
	}
	if config.UserEmail != "demo@example.com" {
		t.Fatalf("email = %q, want normalized email", config.UserEmail)
	}
	if !config.OnlineDemoEnabled || config.UserPassword != "demo-password" {
		t.Fatal("online demo credentials were not loaded")
	}
	if config.StartDate.Format(time.DateOnly) != "2026-05-06" {
		t.Fatalf("start date = %s, want 2026-05-06", config.StartDate.Format(time.DateOnly))
	}
}

func TestLoadConfigRequiresCredentialPair(t *testing.T) {
	t.Setenv("DEMO_DATA_GENERATOR_ENABLED", "")
	t.Setenv("ONLINE_DEMO_USER_EMAIL", "demo@example.com")
	t.Setenv("ONLINE_DEMO_USER_PASSWORD", "")
	if _, err := LoadConfig(t.Context()); err == nil {
		t.Fatal("expected email without password to fail")
	}

	t.Setenv("ONLINE_DEMO_USER_EMAIL", "")
	t.Setenv("ONLINE_DEMO_USER_PASSWORD", "demo-password")
	if _, err := LoadConfig(t.Context()); err == nil {
		t.Fatal("expected password without email to fail")
	}
}
