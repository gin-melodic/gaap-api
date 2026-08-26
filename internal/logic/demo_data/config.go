package demo_data

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/gogf/gf/v2/errors/gerror"
)

const (
	defaultStartDate              = "2026-04-02"
	defaultInitialBackfillEndDate = "2026-08-23"
	defaultTimezone               = "America/Los_Angeles"
)

type Config struct {
	Enabled                bool
	OnlineDemoEnabled      bool
	UserEmail              string
	UserPassword           string
	StartDate              time.Time
	InitialBackfillEndDate time.Time
	Location               *time.Location
}

func LoadConfig(_ context.Context) (Config, error) {
	enabled := strings.EqualFold(strings.TrimSpace(os.Getenv("DEMO_DATA_GENERATOR_ENABLED")), "true")
	locationName := strings.TrimSpace(os.Getenv("DEMO_DATA_TIMEZONE"))
	if locationName == "" {
		locationName = defaultTimezone
	}
	location, err := time.LoadLocation(locationName)
	if err != nil {
		return Config{}, gerror.Wrap(err, "invalid demo data timezone")
	}

	startDateText := strings.TrimSpace(os.Getenv("DEMO_DATA_START_DATE"))
	if startDateText == "" {
		startDateText = defaultStartDate
	}
	startDate, err := time.ParseInLocation(time.DateOnly, startDateText, location)
	if err != nil {
		return Config{}, gerror.Wrap(err, "invalid demo data start date")
	}
	initialBackfillEndDate, err := time.ParseInLocation(time.DateOnly, defaultInitialBackfillEndDate, location)
	if err != nil {
		return Config{}, gerror.Wrap(err, "invalid initial demo backfill end date")
	}

	userEmail := strings.ToLower(strings.TrimSpace(os.Getenv("ONLINE_DEMO_USER_EMAIL")))
	userPassword := os.Getenv("ONLINE_DEMO_USER_PASSWORD")
	hasEmail := userEmail != ""
	hasPassword := strings.TrimSpace(userPassword) != ""
	if hasEmail != hasPassword {
		return Config{}, gerror.New("ONLINE_DEMO_USER_EMAIL and ONLINE_DEMO_USER_PASSWORD must be configured together")
	}
	if enabled && !hasEmail {
		return Config{}, gerror.New("online demo credentials are required when demo data generation is enabled")
	}

	return Config{
		Enabled:                enabled,
		OnlineDemoEnabled:      hasEmail && hasPassword,
		UserEmail:              userEmail,
		UserPassword:           userPassword,
		StartDate:              startDate,
		InitialBackfillEndDate: initialBackfillEndDate,
		Location:               location,
	}, nil
}
