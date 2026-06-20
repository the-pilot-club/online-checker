package health

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// at returns a Probe whose clock is driven by clk, for deterministic tests.
func at(threshold time.Duration, clk *time.Time) *Probe {
	p := New(threshold)
	p.now = func() time.Time { return *clk }
	return p
}

func TestStale_FreshTaskIsLive(t *testing.T) {
	now := time.Unix(1000, 0)
	p := at(30*time.Second, &now)
	p.Register("atc")

	now = now.Add(10 * time.Second)
	if stale := p.Stale(); len(stale) != 0 {
		t.Fatalf("expected no stale tasks, got %v", stale)
	}
}

func TestStale_DetectsWedgedTask(t *testing.T) {
	now := time.Unix(1000, 0)
	p := at(30*time.Second, &now)
	p.Register("atc")
	p.Register("pilot")

	// pilot keeps beating; atc goes silent.
	now = now.Add(31 * time.Second)
	p.Beat("pilot")

	stale := p.Stale()
	if len(stale) != 1 || stale[0] != "atc" {
		t.Fatalf("expected [atc] stale, got %v", stale)
	}
}

func TestLivenessHandler(t *testing.T) {
	now := time.Unix(1000, 0)
	p := at(30*time.Second, &now)
	p.Register("atc")

	if code := get(p.Handler(), "/health"); code != http.StatusOK {
		t.Errorf("fresh task: got %d, want 200", code)
	}

	now = now.Add(31 * time.Second)
	if code := get(p.Handler(), "/health"); code != http.StatusServiceUnavailable {
		t.Errorf("wedged task: got %d, want 503", code)
	}
}

func TestReadinessHandler(t *testing.T) {
	now := time.Unix(1000, 0)
	p := at(30*time.Second, &now)

	if code := get(p.Handler(), "/ready"); code != http.StatusServiceUnavailable {
		t.Errorf("before ready: got %d, want 503", code)
	}

	p.SetReady(true)
	if code := get(p.Handler(), "/ready"); code != http.StatusOK {
		t.Errorf("after ready: got %d, want 200", code)
	}

	p.SetReady(false)
	if code := get(p.Handler(), "/ready"); code != http.StatusServiceUnavailable {
		t.Errorf("after un-ready: got %d, want 503", code)
	}
}

func get(h http.Handler, path string) int {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code
}
