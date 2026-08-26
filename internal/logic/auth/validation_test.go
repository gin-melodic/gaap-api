package auth

import (
	"context"
	"strings"
	"testing"

	"gaap-api/internal/model"
)

func TestValidateRegistrationInput(t *testing.T) {
	tests := []struct {
		name        string
		input       model.RegisterInput
		wantError   bool
		wantMessage string
	}{
		{
			name:  "valid",
			input: model.RegisterInput{Email: "user+beta@example.com", Password: "StrongPass123", Nickname: " Beta User "},
		},
		{
			name:  "unicode password",
			input: model.RegisterInput{Email: "unicode@example.com", Password: "密码测试12345", Nickname: "用户"},
		},
		{
			name:      "invalid email",
			input:     model.RegisterInput{Email: "invalid-email", Password: "StrongPass123", Nickname: "User"},
			wantError: true,
		},
		{
			name:        "oversized email",
			input:       model.RegisterInput{Email: strings.Repeat("a", 250) + "@example.com", Password: "StrongPass123", Nickname: "User"},
			wantError:   true,
			wantMessage: "email must not exceed 255 characters",
		},
		{
			name:      "weak password",
			input:     model.RegisterInput{Email: "weak@example.com", Password: "123456", Nickname: "User"},
			wantError: true,
		},
		{
			name:      "oversized password",
			input:     model.RegisterInput{Email: "long@example.com", Password: strings.Repeat("x", 101), Nickname: "User"},
			wantError: true,
		},
		{
			name:      "empty nickname",
			input:     model.RegisterInput{Email: "nickname@example.com", Password: "StrongPass123", Nickname: "   "},
			wantError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := test.input
			err := validateRegistrationInput(&input)
			if (err != nil) != test.wantError {
				t.Fatalf("validateRegistrationInput() error = %v, wantError %t", err, test.wantError)
			}
			if test.wantMessage != "" && (err == nil || err.Error() != test.wantMessage) {
				t.Fatalf("validateRegistrationInput() error = %v, want %q", err, test.wantMessage)
			}
			if !test.wantError && input.Nickname != strings.TrimSpace(test.input.Nickname) {
				t.Fatalf("nickname was not normalized: %q", input.Nickname)
			}
		})
	}
}

func TestProductionLoginRequiresTurnstile(t *testing.T) {
	t.Setenv("GF_ENV", "production")
	t.Setenv("ENV", "")
	_, err := New().Login(context.Background(), model.LoginInput{
		Email:    "user@example.com",
		Password: "StrongPass123",
	})
	if err == nil || err.Error() != "invalid email or password" {
		t.Fatalf("production login without Turnstile returned %v", err)
	}
}

func TestDemoLoginRequiresConfiguredCredentialPair(t *testing.T) {
	t.Setenv("ONLINE_DEMO_USER_EMAIL", "")
	t.Setenv("ONLINE_DEMO_USER_PASSWORD", "")
	if _, err := New().DemoLogin(context.Background()); err == nil || err.Error() != "demo login unavailable" {
		t.Fatalf("demo login without credentials returned %v", err)
	}
}
