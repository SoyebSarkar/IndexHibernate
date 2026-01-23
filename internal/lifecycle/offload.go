package lifecycle

import (
	"log"
	"path/filepath"

	"github.com/SoyebSarkar/Hiberstack/internal/metrics"
	"github.com/SoyebSarkar/Hiberstack/internal/state"
	"github.com/SoyebSarkar/Hiberstack/snapshot"
)

func (m *Manager) Offload(collection string) error {
	st := m.stateStore.Get(collection)
	if st != state.Draining {
		return nil
	}
	log.Printf("lifecycle offload start collection=%s", collection)
	baseDir := filepath.Join(m.snapshotDir, collection)

	schema, err := m.ts.GetSchema(collection)
	if err != nil {
		return err
	}

	if err := snapshot.SaveSchema(baseDir, schema); err != nil {
		return err
	}

	docs, err := m.ts.Export(collection)
	if err != nil {
		return err
	}
	defer docs.Close()

	if err := snapshot.SaveDocuments(baseDir, docs); err != nil {
		return err
	}

	if err := m.ts.Delete(collection); err != nil {
		return err
	}

	m.stateStore.Set(collection, state.Cold)
	metrics.OffloadTotal.Inc()

	return nil
}
