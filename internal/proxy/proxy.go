package proxy

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/SoyebSarkar/Hiberstack/internal/config"
	"github.com/SoyebSarkar/Hiberstack/internal/lifecycle"
	"github.com/SoyebSarkar/Hiberstack/internal/metrics"
	"github.com/SoyebSarkar/Hiberstack/internal/state"
)

type Reloader interface {
	Reload(collection string)
}

type Proxy struct {
	rp           *httputil.ReverseProxy
	lifecycleMgr *lifecycle.Manager
	reloadMode   config.ReloadMode
	stateStore   *state.Store
	inflight     sync.Map
}

func New(
	target string,
	lifecycleMgr *lifecycle.Manager,
	stateStore *state.Store,
	reloadMode config.ReloadMode,
) (*Proxy, error) {

	u, err := url.Parse(target)
	if err != nil {
		return nil, err
	}

	p := &Proxy{
		lifecycleMgr: lifecycleMgr,
		stateStore:   stateStore,
		reloadMode:   reloadMode,
	}

	rp := httputil.NewSingleHostReverseProxy(u)

	// MODIFY RESPONSE: async reload only
	rp.ModifyResponse = func(resp *http.Response) error {
		collection := extractCollection(resp.Request.URL.Path)
		if collection == "" {
			return nil
		}

		if resp.StatusCode < 400 {
			p.stateStore.Touch(collection)
			return nil
		}

		// Handle 404 Not Found
		if resp.StatusCode != http.StatusNotFound {
			return nil
		}

		// If we have never seen this collection, pass through Typesense 404
		if !p.stateStore.Exists(collection) {
			return nil
		}

		current := p.stateStore.Get(collection)

		if current == state.Cold && p.reloadMode == config.ReloadAsync {
			log.Println("async cold reload triggered:", collection)

			ch, loaded := p.inflight.LoadOrStore(collection, make(chan struct{}))
			done := ch.(chan struct{})

			if !loaded {
				go func() {
					p.lifecycleMgr.Reload(collection)
					close(done)
					p.inflight.Delete(collection)
				}()
			}

			return replaceWithWarming(resp)
		}

		// Already loading → tell client to retry
		if current == state.Loading {
			return replaceWithWarming(resp)
		}

		// Any other case → pass through
		return nil
	}

	p.rp = rp
	return p, nil
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	collection := extractCollection(r.URL.Path)

	// No collection → pass through
	if collection == "" {
		p.rp.ServeHTTP(w, r)
		return
	}

	// --------------------
	// WRITE REQUESTS
	// --------------------
	if isWriteRequest(r) {
		st := p.stateStore.Get(collection)

		// Block writes during draining
		if st == state.Draining {
			metrics.WriteBlockedTotal.Inc()
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte(`{"message":"collection is draining, writes are temporarily disabled"}`))
			return
		}
		// Async reload mode → ModifyResponse handles reload
		if p.reloadMode != config.ReloadBlocking {
			p.rp.ServeHTTP(w, r)
			return
		}

		// COLD write → trigger async reload
		if st == state.Cold {
			ch, loaded := p.inflight.LoadOrStore(collection, make(chan struct{}))
			done := ch.(chan struct{})

			if !loaded {
				go func() {
					log.Printf("async reload triggered by write collection=%s", collection)
					p.lifecycleMgr.Reload(collection)
					close(done)
					p.inflight.Delete(collection)
				}()
			}
		}

		// Forward write
		rr := httptest.NewRecorder()
		p.rp.ServeHTTP(rr, r)

		// In blocking mode, hide misleading 404s
		if st == state.Cold && rr.Code == http.StatusNotFound {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"message":"collection warming up"}`))
			return
		}

		copyResponse(w, rr)
		return
	}

	// --------------------
	// READ REQUESTS
	// --------------------

	// Async reload mode → ModifyResponse handles reload
	if p.reloadMode != config.ReloadBlocking {
		p.rp.ServeHTTP(w, r)
		return
	}

	st := p.stateStore.Get(collection)

	// Non-cold reads → pass through
	if st != state.Cold {
		p.rp.ServeHTTP(w, r)
		return
	}

	// --------------------
	// BLOCKING RELOAD (READ + COLD)
	// --------------------

	// First attempt
	rr := httptest.NewRecorder()
	p.rp.ServeHTTP(rr, r)

	// Not a cold miss → return response
	if rr.Code != http.StatusNotFound || !p.stateStore.Exists(collection) {
		copyResponse(w, rr)
		return
	}

	log.Println("blocking reload triggered:", collection)

	ch, loaded := p.inflight.LoadOrStore(collection, make(chan struct{}))
	done := ch.(chan struct{})

	if !loaded {
		go func() {
			p.lifecycleMgr.Reload(collection)
			close(done)
			p.inflight.Delete(collection)
		}()
	}

	start := time.Now()

	select {
	case <-done:
		metrics.BlockingReloadWait.Observe(time.Since(start).Seconds())
		log.Printf("blocking reload completed collection=%s", collection)

		rr2 := httptest.NewRecorder()
		p.rp.ServeHTTP(rr2, r)
		copyResponse(w, rr2)
		return

	case <-time.After(3 * time.Second):
		metrics.BlockingReloadWait.Observe(time.Since(start).Seconds())
		log.Printf("blocking reload timeout collection=%s", collection)
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"message":"collection warming up"}`))
		return
	}
}

// -------------------------
// Helpers
// -------------------------

func extractCollection(path string) string {
	// Expected: /collections/{name}/...
	parts := strings.Split(path, "/")
	if len(parts) >= 3 && parts[1] == "collections" {
		return parts[2]
	}
	return ""
}

func replaceWithWarming(resp *http.Response) error {
	resp.StatusCode = http.StatusServiceUnavailable
	resp.Status = "503 Service Unavailable"
	resp.Body = io.NopCloser(
		bytes.NewBufferString(`{"message":"collection warming up, retry shortly"}`),
	)
	resp.Header.Set("Content-Type", "application/json")
	resp.Header.Set("Retry-After", "2")
	return nil
}

func isWriteRequest(r *http.Request) bool {
	switch r.Method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func copyResponse(w http.ResponseWriter, rr *httptest.ResponseRecorder) {
	for k, v := range rr.Header() {
		w.Header()[k] = v
	}
	w.WriteHeader(rr.Code)
	w.Write(rr.Body.Bytes())
}
