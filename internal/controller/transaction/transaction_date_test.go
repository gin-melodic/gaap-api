package transaction

import (
	"testing"
	"time"

	"gaap-api/internal/model/entity"

	"github.com/gogf/gf/v2/os/gtime"
)

var cstZone = time.FixedZone("CST", 8*60*60)

func mustGtime(t *testing.T, value time.Time) *gtime.Time {
	t.Helper()
	return gtime.New(value)
}

// Transaction dates keep full date/time semantics end-to-end: clients send a
// wall-clock value down to seconds and the response must return it verbatim as
// an RFC3339 timestamp instead of truncating it to a bare date.
func TestGtimeToTimestampStringBoundary(t *testing.T) {
	tests := []struct {
		name  string
		input time.Time
		want  string
	}{
		{
			name:  "midnight start of year keeps seconds and offset",
			input: time.Date(2026, time.January, 1, 0, 0, 0, 0, cstZone),
			want:  "2026-01-01T00:00:00+08:00",
		},
		{
			name:  "end of year keeps hour minute second and offset",
			input: time.Date(2026, time.December, 31, 23, 59, 59, 0, cstZone),
			want:  "2026-12-31T23:59:59+08:00",
		},
		{
			name:  "sub-second precision is normalized to whole seconds",
			input: time.Date(2026, 6, 15, 12, 34, 56, 987654321, cstZone),
			want:  "2026-06-15T12:34:56+08:00",
		},
		{
			name:  "zero time boundary serializes deterministically",
			input: time.Date(1, time.January, 1, 0, 0, 0, 0, cstZone),
			want:  "0001-01-01T00:00:00+08:00",
		},
		{
			name:  "utc instant is normalized to the canonical zone",
			input: time.Date(2026, 3, 5, 9, 7, 8, 0, time.UTC),
			want:  "2026-03-05T17:07:08+08:00",
		},
		{
			name:  "utc instant crossing the local day boundary keeps the shifted date prefix",
			input: time.Date(2026, 9, 1, 16, 0, 0, 0, time.UTC),
			want:  "2026-09-02T00:00:00+08:00",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := gtimeToTimestampString(mustGtime(t, tt.input)); got != tt.want {
				t.Fatalf("gtimeToTimestampString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGtimeToTimestampStringNilBoundary(t *testing.T) {
	if got := gtimeToTimestampString(nil); got != "" {
		t.Fatalf("gtimeToTimestampString(nil) = %q, want empty string", got)
	}
}

// The create/update input path parses "YYYY-MM-DD HH:mm:ss" payloads with
// gtime.NewFromStr; the parsed value must round-trip through the serializer
// without losing any sub-day component.
func TestClientDateTimePayloadRoundTrip(t *testing.T) {
	// gtime.NewFromStr parses naive wall-clock strings in time.Local; production
	// runs with TZ=Asia/Shanghai, so pin the test to the canonical +08 zone to
	// stay independent of the CI runner's timezone.
	origLocal := time.Local
	time.Local = cstZone
	t.Cleanup(func() { time.Local = origLocal })

	payloads := []struct {
		name    string
		payload string
		want    string
	}{
		{"midnight", "2026-01-01 00:00:00", "2026-01-01T00:00:00+08:00"},
		{"end of year", "2026-12-31 23:59:59", "2026-12-31T23:59:59+08:00"},
	}
	for _, tt := range payloads {
		t.Run(tt.name, func(t *testing.T) {
			parsed := gtime.NewFromStr(tt.payload)
			if parsed == nil {
				t.Fatalf("gtime.NewFromStr(%q) returned nil", tt.payload)
			}
			got := gtimeToTimestampString(parsed)
			want, err := time.Parse(time.RFC3339, tt.want)
			if err != nil {
				t.Fatalf("invalid expected timestamp %q: %v", tt.want, err)
			}
			if got != want.Format(time.RFC3339) {
				t.Fatalf("round trip = %q, want %q", got, want.Format(time.RFC3339))
			}
		})
	}
}

func TestEntityToProtoPreservesTimestamp(t *testing.T) {
	date := time.Date(2026, 8, 14, 15, 30, 5, 0, cstZone)
	protoTx := entityToProto(&entity.Transactions{
		Date: mustGtime(t, date),
	})

	if protoTx == nil {
		t.Fatal("entityToProto returned nil")
	}
	parsed, err := time.Parse(time.RFC3339, protoTx.Date)
	if err != nil {
		t.Fatalf("proto Date %q is not valid RFC3339: %v", protoTx.Date, err)
	}
	if !parsed.Equal(date) {
		t.Fatalf("proto Date = %s (%s), want original instant %s", parsed.Format(time.RFC3339Nano), protoTx.Date, date.Format(time.RFC3339Nano))
	}
}

func TestEntitiesToProtosPreservesTimestampSlice(t *testing.T) {
	first := time.Date(2026, 1, 1, 8, 0, 1, 0, cstZone)
	second := time.Date(2026, 12, 31, 23, 59, 59, 0, cstZone)
	result := entitiesToProtos([]entity.Transactions{
		{Date: mustGtime(t, first)},
		{Date: mustGtime(t, second)},
	})
	if len(result) != 2 {
		t.Fatalf("entitiesToProtos length = %d, want 2", len(result))
	}
	for i, tt := range []string{"2026-01-01T08:00:01+08:00", "2026-12-31T23:59:59+08:00"} {
		if result[i].Date != tt {
			t.Fatalf("entitiesToProtos()[%d].Date = %q, want %q", i, result[i].Date, tt)
		}
	}
}
