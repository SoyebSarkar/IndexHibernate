package state

import "sync"

type State string

const (
	Hot     State = "HOT"
	Cold    State = "COLD"
	Loading State = "LOADING"
)

type Store struct {
	mu    sync.RWMutex
	state map[string]State
}

func New() *Store {
	return &Store{
		state: make(map[string]State),
	}
}

func (s *Store) Get(collection string) State {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state[collection]
}

func (s *Store) Set(collection string, st State) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state[collection] = st
}

func (s *Store) Exists(collection string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.state[collection]
	return ok
}
