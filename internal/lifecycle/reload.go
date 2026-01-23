package lifecycle

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/SoyebSarkar/Hiberstack/internal/metrics"
	"github.com/SoyebSarkar/Hiberstack/internal/state"
)

func (m *Manager) Reload(collection string) error {
	st := m.stateStore.Get(collection)
	if st != state.Cold {
		return nil
	}
	m.reloadSem <- struct{}{}
	defer func() {
		<-m.reloadSem
	}()
	start := time.Now()
	log.Printf("lifecycle reload start collection=%s", collection)
	m.stateStore.Set(collection, state.Loading)

	baseDir := filepath.Join(m.snapshotDir, collection)

	schema, err := os.ReadFile(filepath.Join(baseDir, "schema.json"))
	if err != nil {
		return err
	}

	if err := m.ts.CreateCollection(schema); err != nil {
		return err
	}

	file, err := os.Open(filepath.Join(baseDir, "documents.jsonl"))
	if err != nil {
		return err
	}
	defer file.Close()

	if err := m.ts.ImportDocuments(collection, file); err != nil {
		return err
	}

	m.stateStore.Set(collection, state.Hot)
	log.Printf("lifecycle reload complete collection=%s duration=%s", collection, time.Since(start))
	metrics.ReloadTotal.Inc()
	metrics.ReloadDuration.Observe(time.Since(start).Seconds())
	return nil
}
