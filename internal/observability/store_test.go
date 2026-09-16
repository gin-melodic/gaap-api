package observability

import (
	"sync"
	"testing"
	"time"

	"gaap-api/internal/model"
)

func TestRecordAndCounter(t *testing.T) {
	ResetForTest()
	Record("test.counter")
	Record("test.counter")
	if got := Counter("test.counter"); got != 2 {
		t.Fatalf("Counter(test.counter) = %d, want 2", got)
	}
	if got := Counter("test.other"); got != 0 {
		t.Fatalf("Counter(test.other) = %d, want 0", got)
	}
}

func TestWindowSince(t *testing.T) {
	ResetForTest()
	now := time.Date(2026, 9, 10, 12, 30, 45, 0, time.UTC)

	// Current minute.
	RecordAt("win", now)
	RecordAt("win", now)
	// Previous minute.
	RecordAt("win", now.Add(-time.Minute))
	// Two minutes ago (outside a 1-minute window, inside 2).
	RecordAt("win", now.Add(-2*time.Minute))
	// Old bucket outside any reasonable window.
	RecordAt("win", now.Add(-3*time.Hour))

	if got := sinceAt("win", 0, now); got != 2 {
		t.Fatalf("sinceAt(win, 0) = %d, want 2", got)
	}
	if got := sinceAt("win", 1, now); got != 3 {
		t.Fatalf("sinceAt(win, 1) = %d, want 3", got)
	}
	if got := sinceAt("win", 2, now); got != 4 {
		t.Fatalf("sinceAt(win, 2) = %d, want 4", got)
	}
}

func TestWindowSlotReuseAfterCycle(t *testing.T) {
	ResetForTest()
	base := time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC)
	// Fill a slot at minute 0.
	RecordAt("cycle", base)
	// The same slot index reappears after windowBuckets minutes (2h later);
	// the stale bucket must be reset, not summed.
	later := base.Add(time.Duration(windowBuckets) * time.Minute)
	RecordAt("cycle", later)

	if got := sinceAt("cycle", 0, later); got != 1 {
		t.Fatalf("sinceAt(cycle, 0) after slot cycle = %d, want 1", got)
	}
}

func TestRecentEventsRingBufferEvictsOldest(t *testing.T) {
	ResetForTest()
	for i := 0; i < maxRecentEvents+25; i++ {
		RecordEvent("evt", "info", "")
	}
	all := RecentEvents(maxRecentEvents)
	if len(all) != maxRecentEvents {
		t.Fatalf("RecentEvents = %d, want %d", len(all), maxRecentEvents)
	}
	small := RecentEvents(5)
	if len(small) != 5 {
		t.Fatalf("RecentEvents(5) = %d, want 5", len(small))
	}
	// Chronological order: first <= last.
	if !small[0].Time.Before(small[4].Time) && !small[0].Time.Equal(small[4].Time) {
		t.Fatalf("events out of chronological order")
	}
	_ = all
}

func TestRecordHTTPStatusClasses(t *testing.T) {
	ResetForTest()
	RecordHTTP("/ok", 200)
	RecordHTTP("/redirect", 301)
	RecordHTTP("/bad", 404)
	RecordHTTP("/boom", 502)
	RecordHTTP("/unset", 0)

	want := map[string]int64{
		"http.total": 5,
		"http.2xx":   2,
		"http.3xx":   1,
		"http.4xx":   1,
		"http.5xx":   1,
	}
	for name, expected := range want {
		if got := Counter(name); got != expected {
			t.Fatalf("Counter(%s) = %d, want %d", name, got, expected)
		}
	}
	events := RecentEvents(maxRecentEvents)
	if len(events) != 1 {
		t.Fatalf("expected exactly one 5xx ring entry, got %d", len(events))
	}
	if events[0].Event != "http.5xx" || events[0].Detail != "/boom 502" || events[0].Level != "error" {
		t.Fatalf("unexpected 5xx event entry: %+v", events[0])
	}
}

func TestReconSnapshotRoundTrip(t *testing.T) {
	ResetForTest()
	if got := LastReconciliation(); got != nil {
		t.Fatalf("LastReconciliation() = %+v, want nil", got)
	}
	report := &model.Report{Passed: true, AccountsChecked: 3, TransactionsChecked: 9}
	at := time.Now()
	SetLastReconciliation(report, "manual", at)

	got := LastReconciliation()
	if got == nil || got.Source != "manual" || !got.At.Equal(at) || got.Report != report {
		t.Fatalf("unexpected snapshot: %+v", got)
	}
}

func TestConcurrentRecording(t *testing.T) {
	ResetForTest()
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				Record("conc")
				RecordEvent("conc.evt", "info", "")
			}
		}()
	}
	wg.Wait()
	if got := Counter("conc"); got != 8000 {
		t.Fatalf("Counter(conc) = %d, want 8000", got)
	}
	if got := len(RecentEvents(maxRecentEvents)); got != maxRecentEvents {
		t.Fatalf("RecentEvents = %d, want %d", got, maxRecentEvents)
	}
}
