package proxy

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"

	"github.com/SoyebSarkar/Hiberstack/internal/state"
)

type Reloader interface {
	Reload(collection string)
}

type Proxy struct {
	rp         *httputil.ReverseProxy
	reloader   Reloader
	stateStore *state.Store
	inflight   sync.Map 
}

func New(
	target string,
	reloader Reloader,
	stateStore *state.Store,
) (*Proxy, error) {
	u, err := url.Parse(target)
	if err != nil {
		return nil, err
	}

	p := &Proxy{
		reloader:   reloader,
		stateStore: stateStore,
	}

	rp := httputil.NewSingleHostReverseProxy(u)

	rp.ModifyResponse = func(resp *http.Response) error {
		// Only care about 404s
		if resp.StatusCode != http.StatusNotFound {
			return nil
		}

		collection := extractCollection(resp.Request.URL.Path)
		if collection == "" {
			return nil
		}

		// If we have never seen this collection, pass through Typesense 404
		if !p.stateStore.Exists(collection) {
			return nil
		}

		current := p.stateStore.Get(collection)

		// Cold → trigger reload
		if current == state.Cold {
			log.Println("cold collection hit:", collection)

			p.stateStore.Set(collection, state.Loading)

			if _, loaded := p.inflight.LoadOrStore(collection, true); !loaded {
				go func() {
					p.reloader.Reload(collection)
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
	p.rp.ServeHTTP(w, r)
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
