package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/SoyebSarkar/Hiberstack/internal/engine/typesense"
	"github.com/SoyebSarkar/Hiberstack/snapshot"
)

func registerAdmin(
	mux *http.ServeMux,
	ts *typesense.Client,
	snapshotDir string,
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

		baseDir := filepath.Join(snapshotDir, collection)

		schema, err := os.ReadFile(filepath.Join(baseDir, "schema.json"))
		if err != nil {
			http.Error(w, "schema not found", http.StatusNotFound)
			return
		}

		if err := ts.CreateCollection(schema); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		file, err := os.Open(filepath.Join(baseDir, "documents.jsonl"))
		if err != nil {
			http.Error(w, "documents not found", http.StatusNotFound)
			return
		}
		defer file.Close()

		if err := ts.ImportDocuments(collection, file); err != nil {
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

		baseDir := filepath.Join(snapshotDir, collection)

		// 1️⃣ Get schema
		schema, err := ts.GetSchema(collection)
		if err != nil {
			http.Error(w, "failed to fetch schema: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// 2️⃣ Save schema
		if err := snapshot.SaveSchema(baseDir, schema); err != nil {
			http.Error(w, "failed to save schema: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// 3️⃣ Export documents
		docs, err := ts.Export(collection)
		if err != nil {
			http.Error(w, "export failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer docs.Close()

		// 4️⃣ Save documents
		if err := snapshot.SaveDocuments(baseDir, docs); err != nil {
			http.Error(w, "failed to save documents: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// 5️⃣ Delete collection (RAM freed)
		if err := ts.Delete(collection); err != nil {
			http.Error(w, "delete failed: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Write([]byte("collection offloaded\n"))
	})
}
