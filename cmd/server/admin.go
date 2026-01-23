package main

import (
	"net/http"
	"strings"

	"github.com/SoyebSarkar/Hiberstack/internal/lifecycle"
	"github.com/SoyebSarkar/Hiberstack/internal/state"
)

func registerAdmin(
	mux *http.ServeMux,
	lifecycleMgr *lifecycle.Manager,
	stateStore *state.Store,
) {
	mux.HandleFunc("/admin/reload/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		collection := strings.TrimPrefix(r.URL.Path, "/admin/reload/")
		if collection == "" {
			http.Error(w, "collection name required", http.StatusBadRequest)
			return
		}
		if err := lifecycleMgr.Reload(collection); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Write([]byte("collection reloaded\n"))
	})
	mux.HandleFunc("/admin/offload/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		collection := strings.TrimPrefix(r.URL.Path, "/admin/offload/")
		if collection == "" {
			http.Error(w, "collection name required", http.StatusBadRequest)
			return
		}
		stateStore.Set(collection, state.Draining)
		if err := lifecycleMgr.Offload(collection); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write([]byte("collection offloaded\n"))
	})
}
