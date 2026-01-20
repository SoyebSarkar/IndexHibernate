package main

import (
	"log"
	"net/http"

	"github.com/SoyebSarkar/Hiberstack/internal/config"
	"github.com/SoyebSarkar/Hiberstack/internal/engine/typesense"
	"github.com/SoyebSarkar/Hiberstack/internal/proxy"
)

func main() {
	cfg := config.Load()

	p, err := proxy.New(cfg.Typesense.URL)
	if err != nil {
		log.Fatal(err)
	}
	ts := typesense.New(cfg.Typesense.URL, cfg.Typesense.APIKey)

	mux := http.NewServeMux()

	// 1️⃣ Register admin routes FIRST
	registerAdmin(mux, ts, cfg.SnapshotDir)

	// 2️⃣ Attach proxy as fallback
	mux.Handle("/", p)
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
