package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	_ "github.com/joho/godotenv/autoload"
	"github.com/the-pilot-club/online-checker/internal/functions"
	"github.com/the-pilot-club/online-checker/internal/health"
	"github.com/the-pilot-club/online-checker/internal/store"
	"github.com/the-pilot-club/tpcgo"
)

const (
	checkInterval = 15 * time.Second
	sessionTTL    = 24 * time.Hour
	// livenessThreshold is how long a checker may go without completing an
	// iteration before liveness reports unhealthy. It must comfortably exceed
	// checkInterval so a normal poll cycle is never flagged.
	livenessThreshold = 4 * checkInterval
	defaultHealthAddr = ":8080"
)

func main() {
	atcEnabled := envBool("ENABLE_ATC", true)
	pilotEnabled := envBool("ENABLE_PILOT", true)
	if !atcEnabled && !pilotEnabled {
		log.Fatal("no checkers enabled: set ENABLE_ATC and/or ENABLE_PILOT")
	}

	// Cancel the root context on SIGINT/SIGTERM so the loops drain and exit.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	s, err := tpcgo.NewSession(tpcgo.SessionConfig{
		FCPEnv: "production",
	})
	if err != nil {
		log.Fatalf("failed to create tpcgo session: %v", err)
	}

	probe := health.New(livenessThreshold)
	healthSrv := startHealthServer(probe)

	var wg sync.WaitGroup

	if atcEnabled {
		st, err := store.NewRedis[functions.ATCSession]("online-atc:", sessionTTL)
		if err != nil {
			log.Fatalf("failed to create ATC session store: %v", err)
		}
		defer st.Close()

		probe.Register("atc")
		wg.Add(1)
		go func() {
			defer wg.Done()
			runLoop(ctx, "ATC online checker", func() {
				functions.ATCOnlineCheck(s, st)
				probe.Beat("atc")
			})
		}()
	}

	if pilotEnabled {
		st, err := store.NewRedis[functions.PilotSession]("online-pilots:", sessionTTL)
		if err != nil {
			log.Fatalf("failed to create pilot session store: %v", err)
		}
		defer st.Close()

		probe.Register("pilot")
		wg.Add(1)
		go func() {
			defer wg.Done()
			runLoop(ctx, "pilot online checker", func() {
				functions.OnlineCheck(s, st)
				probe.Beat("pilot")
			})
		}()
	}

	probe.SetReady(true)

	wg.Wait()
	log.Println("checkers stopped; shutting down")

	probe.SetReady(false)
	shutdownHealthServer(healthSrv)
}

// startHealthServer serves the liveness/readiness probes in the background.
func startHealthServer(probe *health.Probe) *http.Server {
	srv := &http.Server{
		Addr:    envString("HEALTH_ADDR", defaultHealthAddr),
		Handler: probe.Handler(),
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("health server error: %v", err)
		}
	}()
	return srv
}

// shutdownHealthServer stops the health server, allowing in-flight probes to
// finish.
func shutdownHealthServer(srv *http.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("health server shutdown error: %v", err)
	}
}

// runLoop runs check on checkInterval until ctx is cancelled.
func runLoop(ctx context.Context, name string, check func()) {
	for {
		log.Printf("starting %s process", name)
		check()
		log.Printf("%s process complete; awaiting datafeed update", name)

		select {
		case <-ctx.Done():
			log.Printf("%s stopping", name)
			return
		case <-time.After(checkInterval):
		}
	}
}

// envBool reads a boolean environment variable, falling back to def when unset
// or unparseable.
func envBool(key string, def bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		log.Printf("invalid %s=%q, using default %v", key, v, def)
		return def
	}
	return b
}

// envString reads a string environment variable, falling back to def when
// unset or empty.
func envString(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
