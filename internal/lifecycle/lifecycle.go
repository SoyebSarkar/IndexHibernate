package lifecycle

import (
	"github.com/SoyebSarkar/Hiberstack/internal/engine/typesense"
	"github.com/SoyebSarkar/Hiberstack/internal/state"
)

type Manager struct {
	ts          *typesense.Client
	snapshotDir string
	stateStore  *state.Store
	reloadSem   chan struct{}
}

func New(
	ts *typesense.Client,
	snapshotDir string,
	stateStore *state.Store,
	maxConcurrentReloads int,
) *Manager {
	return &Manager{
		ts:          ts,
		snapshotDir: snapshotDir,
		stateStore:  stateStore,
		reloadSem:   make(chan struct{}, maxConcurrentReloads),
	}
}
