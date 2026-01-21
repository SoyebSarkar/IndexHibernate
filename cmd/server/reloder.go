package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/SoyebSarkar/Hiberstack/internal/engine/typesense"
	"github.com/SoyebSarkar/Hiberstack/internal/state"
)

type Reloader struct {
	ts          *typesense.Client
	snapshotDir string
	stateStore  *state.Store
}

func (r *Reloader) Reload(collection string) {
	base := filepath.Join(r.snapshotDir, collection)

	schema, err := os.ReadFile(filepath.Join(base, "schema.json"))
	if err != nil {
		log.Println("reload failed (schema):", err)
		return
	}

	if err := r.ts.CreateCollection(schema); err != nil {
		log.Println("reload failed (create):", err)
		return
	}

	file, err := os.Open(filepath.Join(base, "documents.jsonl"))
	if err != nil {
		log.Println("reload failed (docs):", err)
		return
	}
	defer file.Close()

	if err := r.ts.ImportDocuments(collection, file); err != nil {
		log.Println("reload failed (import):", err)
		return
	}
	r.stateStore.Set(collection, state.Hot)

	log.Println("auto reload completed:", collection)
}
