// Package health exposes Kubernetes-style liveness and readiness checks.
//
// Liveness is driven by per-task heartbeats: each background loop records a
// beat once per iteration, and the probe reports unhealthy if any task's last
// beat is older than a staleness threshold (i.e. the loop has wedged).
// Readiness is a simple flag flipped once start-up wiring is complete.
package health

import (
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// Probe tracks task heartbeats and answers liveness/readiness HTTP checks.
// It is safe for concurrent use.
type Probe struct {
	mu        sync.Mutex
	beats     map[string]time.Time
	threshold time.Duration
	ready     bool
	now       func() time.Time
}

// New returns a Probe that considers a task stale when its last heartbeat is
// older than threshold.
func New(threshold time.Duration) *Probe {
	return &Probe{
		beats:     make(map[string]time.Time),
		threshold: threshold,
		now:       time.Now,
	}
}

// Register starts tracking a task, seeding its heartbeat with the current time
// so it is considered live during start-up.
func (p *Probe) Register(name string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.beats[name] = p.now()
}

// Beat records that the named task made progress.
func (p *Probe) Beat(name string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.beats[name] = p.now()
}

// SetReady marks the probe ready (or not) to serve.
func (p *Probe) SetReady(ready bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ready = ready
}

// Stale returns the names of tasks whose last heartbeat exceeds the threshold,
// sorted for stable output.
func (p *Probe) Stale() []string {
	p.mu.Lock()
	defer p.mu.Unlock()

	cutoff := p.now().Add(-p.threshold)
	var stale []string
	for name, last := range p.beats {
		if last.Before(cutoff) {
			stale = append(stale, name)
		}
	}
	sort.Strings(stale)
	return stale
}

func (p *Probe) isReady() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.ready
}

// Handler returns an http.Handler serving /health (liveness) and /ready
// (readiness).
func (p *Probe) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", p.serveLive)
	mux.HandleFunc("/ready", p.serveReady)
	return mux
}

func (p *Probe) serveLive(w http.ResponseWriter, _ *http.Request) {
	if stale := p.Stale(); len(stale) > 0 {
		http.Error(w, "unhealthy: stale tasks: "+strings.Join(stale, ", "), http.StatusServiceUnavailable)
		return
	}
	writeOK(w)
}

func (p *Probe) serveReady(w http.ResponseWriter, _ *http.Request) {
	if !p.isReady() {
		http.Error(w, "not ready", http.StatusServiceUnavailable)
		return
	}
	writeOK(w)
}

func writeOK(w http.ResponseWriter) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
