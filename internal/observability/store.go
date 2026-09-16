// Package observability provides lightweight in-process metrics and an
// event ring buffer for the admin console. It is deliberately dependency-free
// and process-local: counters reset when the API restarts, which matches the
// "post-deploy observation window" use case.
package observability

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"gaap-api/internal/model"
)

// Counter names. The console UI reads exactly these names.
const (
	EventHTTPTotal = "http.total"
	EventHTTP2xx   = "http.2xx"
	EventHTTP3xx   = "http.3xx"
	EventHTTP4xx   = "http.4xx"
	EventHTTP5xx   = "http.5xx"

	EventALESignatureInvalid       = "ale.signature_invalid"
	EventALETimestampInvalid       = "ale.timestamp_invalid"
	EventALEReplay                 = "ale.replay"
	EventALEDecryptFailed          = "ale.decrypt_failed"
	EventALEMissingHeaders         = "ale.missing_headers"
	EventALEBootstrapUnavailable   = "ale.bootstrap_unavailable"
	EventALEReplayStoreUnavailable = "ale.replay_store_unavailable"
	EventALESessionExpired         = "ale.session_expired"

	EventAuthLoginSuccess   = "auth.login_success"
	EventAuthLoginFailed    = "auth.login_failed"
	EventAuthRefreshSuccess = "auth.refresh_success"
	EventAuthRefreshFailed  = "auth.refresh_failed"

	EventReconciliationStartupPassed = "reconciliation.startup_passed"
	EventReconciliationStartupFailed = "reconciliation.startup_failed"
	EventReconciliationManualPassed  = "reconciliation.manual_passed"
	EventReconciliationManualFailed  = "reconciliation.manual_failed"
)

// ALEEvents lists the ALE/HMAC event counters surfaced by the console.
var ALEEvents = []string{
	EventALESignatureInvalid,
	EventALETimestampInvalid,
	EventALEReplay,
	EventALEDecryptFailed,
	EventALEMissingHeaders,
	EventALEBootstrapUnavailable,
	EventALEReplayStoreUnavailable,
	EventALESessionExpired,
}

// AuthEvents lists the auth/session event counters surfaced by the console.
var AuthEvents = []string{
	EventAuthLoginSuccess,
	EventAuthLoginFailed,
	EventAuthRefreshSuccess,
	EventAuthRefreshFailed,
}

// ReconSnapshot stores the latest reconciliation run for the console.
type ReconSnapshot struct {
	At     time.Time          `json:"at"`
	Source string             `json:"source"`
	Report *model.Report      `json:"report"`
}

const (
	windowBuckets   = 120 // 2 hours of one-minute buckets
	windowBucketLen = time.Minute
	maxRecentEvents = 100
)

type windowBucket struct {
	start  time.Time
	counts map[string]int64
}

var (
	mu       sync.RWMutex
	counters = map[string]*atomic.Int64{}

	windowMu sync.Mutex
	buckets  [windowBuckets]*windowBucket

	eventsMu sync.Mutex
	events   []model.OpsEvent

	reconMu   sync.RWMutex
	lastRecon *ReconSnapshot

	processStart = time.Now()
)

// ProcessStart returns the API process start time.
func ProcessStart() time.Time {
	return processStart
}

// Record increments a cumulative counter and the one-minute window for now.
func Record(name string) {
	RecordAt(name, time.Now())
}

// RecordAt increments a cumulative counter and the one-minute window at a
// specific time (used by tests).
func RecordAt(name string, now time.Time) {
	counterFor(name).Add(1)
	recordWindow(name, now)
}

func counterFor(name string) *atomic.Int64 {
	mu.RLock()
	c, ok := counters[name]
	mu.RUnlock()
	if ok {
		return c
	}
	mu.Lock()
	defer mu.Unlock()
	if c, ok = counters[name]; ok {
		return c
	}
	c = &atomic.Int64{}
	counters[name] = c
	return c
}

func recordWindow(name string, now time.Time) {
	key := now.Truncate(windowBucketLen)
	idx := bucketIndex(key)
	windowMu.Lock()
	defer windowMu.Unlock()
	b := buckets[idx]
	if b == nil || !b.start.Equal(key) {
		b = &windowBucket{start: key, counts: make(map[string]int64)}
		buckets[idx] = b
	}
	b.counts[name]++
}

func bucketIndex(key time.Time) int {
	return int(key.Unix()/int64(windowBucketLen.Seconds())) % windowBuckets
}

// Counter returns the cumulative count for name.
func Counter(name string) int64 {
	return counterFor(name).Load()
}

// Since returns the window count for name over the trailing minutes (inclusive).
func Since(name string, minutes int) int64 {
	return sinceAt(name, minutes, time.Now())
}

func sinceAt(name string, minutes int, now time.Time) int64 {
	var total int64
	for m := 0; m <= minutes; m++ {
		key := now.Add(-time.Duration(m) * windowBucketLen).Truncate(windowBucketLen)
		idx := bucketIndex(key)
		windowMu.Lock()
		if b := buckets[idx]; b != nil && b.start.Equal(key) {
			total += b.counts[name]
		}
		windowMu.Unlock()
	}
	return total
}

// RecordEvent increments the counter for name and appends a ring-buffer entry.
func RecordEvent(name string, level string, detail string) {
	now := time.Now()
	RecordAt(name, now)
	eventsMu.Lock()
	defer eventsMu.Unlock()
	events = append(events, model.OpsEvent{Time: now, Level: level, Event: name, Detail: detail})
	if len(events) > maxRecentEvents {
		events = events[len(events)-maxRecentEvents:]
	}
}

// RecentEvents returns the newest limit events in chronological order.
func RecentEvents(limit int) []model.OpsEvent {
	if limit <= 0 || limit > maxRecentEvents {
		limit = maxRecentEvents
	}
	eventsMu.Lock()
	defer eventsMu.Unlock()
	start := len(events) - limit
	if start < 0 {
		start = 0
	}
	out := make([]model.OpsEvent, len(events)-start)
	copy(out, events[start:])
	return out
}

// SetLastReconciliation stores the latest reconciliation run for the console.
func SetLastReconciliation(report *model.Report, source string, at time.Time) {
	reconMu.Lock()
	defer reconMu.Unlock()
	lastRecon = &ReconSnapshot{At: at, Source: source, Report: report}
}

// LastReconciliation returns the stored reconciliation snapshot (may be nil).
func LastReconciliation() *ReconSnapshot {
	reconMu.RLock()
	defer reconMu.RUnlock()
	return lastRecon
}

// RecordHTTP tallies an HTTP response by status class. 5xx responses are also
// written to the recent-events ring buffer with their path for fast triage.
// A zero status (not yet finalized) is counted as 200.
func RecordHTTP(path string, status int) {
	if status == 0 {
		status = 200
	}
	Record(EventHTTPTotal)
	switch {
	case status >= 500:
		RecordEvent(EventHTTP5xx, "error", fmt.Sprintf("%s %d", path, status))
	case status >= 400:
		Record(EventHTTP4xx)
	case status >= 300:
		Record(EventHTTP3xx)
	default:
		Record(EventHTTP2xx)
	}
}

// ResetForTest clears all state (tests only).
func ResetForTest() {
	mu.Lock()
	counters = map[string]*atomic.Int64{}
	mu.Unlock()
	windowMu.Lock()
	var zero [windowBuckets]*windowBucket
	buckets = zero
	windowMu.Unlock()
	eventsMu.Lock()
	events = nil
	eventsMu.Unlock()
	reconMu.Lock()
	lastRecon = nil
	reconMu.Unlock()
}
