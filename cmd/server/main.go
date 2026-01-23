package main

import (
	"log"
	"net/http"

	"github.com/SoyebSarkar/Hiberstack/internal/config"
	"github.com/SoyebSarkar/Hiberstack/internal/engine/typesense"
	"github.com/SoyebSarkar/Hiberstack/internal/lifecycle"
	"github.com/SoyebSarkar/Hiberstack/internal/proxy"
	"github.com/SoyebSarkar/Hiberstack/internal/scheduler"
	"github.com/SoyebSarkar/Hiberstack/internal/state"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	cfg := config.Load()

	// Initialize Typesense client
	ts := typesense.New(cfg.TypesenseURL, cfg.TypesenseAPIKey)

	// Initialize state store
	stateStore, err := state.NewSQLite(cfg.StateDBPath)
	if err != nil {
		log.Fatal(err)
	}
	// Initialize lifecycle manager
	lifecycleMgr := lifecycle.New(
		ts,
		cfg.SnapshotDir,
		stateStore,
		cfg.MaxConcurrentReloads,
	)

	// Initialize and start scheduler
	scheduler := scheduler.New(
		stateStore,
		lifecycleMgr,
		cfg.OffloadAfter,
		cfg.DrainGracePeriod,
		cfg.SchedulerInterval,
	)
	scheduler.Start()

	// Initialize proxy
	proxy, err := proxy.New(cfg.TypesenseURL, lifecycleMgr, stateStore, cfg.ReloadMode)
	if err != nil {
		log.Fatal(err)
	}

	// Setup HTTP server with admin routes and proxy fallback
	mux := http.NewServeMux()

	// Prometheus metrics endpoint
	mux.Handle("/metrics", promhttp.Handler())

	// 1️⃣ Register admin routes FIRST
	registerAdmin(mux, lifecycleMgr, stateStore)

	// 2️⃣ Attach proxy as fallback
	mux.Handle("/", proxy)
	handler := loggingMiddleware(mux)

	// 3️⃣ Start server with mux
	http.ListenAndServe(":"+cfg.Port, handler)

}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf(
			"%s %s %s",
			r.Method,
			r.URL.Path,
			r.RemoteAddr,
		)
		next.ServeHTTP(w, r)
	})
}
