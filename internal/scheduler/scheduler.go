package scheduler

import (
	"log"
	"time"

	"github.com/SoyebSarkar/Hiberstack/internal/lifecycle"
	"github.com/SoyebSarkar/Hiberstack/internal/metrics"
	"github.com/SoyebSarkar/Hiberstack/internal/state"
)

type Offloader interface {
	Offload(collection string) error
}

type Scheduler struct {
	store        *state.Store
	lifecycleMgr *lifecycle.Manager
	offloadAfter time.Duration
	gracePeriod  time.Duration
	interval     time.Duration
}

func New(
	store *state.Store,
	lifecycleMgr *lifecycle.Manager,
	offloadAfter time.Duration,
	drainGracePeriod time.Duration,
	interval time.Duration,
) *Scheduler {
	return &Scheduler{
		store:        store,
		lifecycleMgr: lifecycleMgr,
		offloadAfter: offloadAfter,
		gracePeriod:  drainGracePeriod,
		interval:     interval,
	}
}

func (s *Scheduler) Start() {
	ticker := time.NewTicker(s.interval)

	go func() {
		for range ticker.C {
			s.runOnce()
			metrics.UpdateStateGauges(s.store)
		}
	}()
}

func (s *Scheduler) runOnce() {
	collections := s.store.ListHotOlderThan(s.offloadAfter)

	for _, c := range collections {
		log.Printf("scheduler marking draining collection=%s idle_for=%s", c, s.offloadAfter.String())
		s.store.Set(c, state.Draining)
		go s.drainAndOffload(c)
	}
}

func (s *Scheduler) drainAndOffload(collection string) {
	time.Sleep(s.gracePeriod)

	// State might have changed
	if s.store.Get(collection) != state.Draining {
		return
	}

	// Activity resumed → cancel offload
	if s.store.WasRecentlyAccessed(collection, s.offloadAfter) {
		log.Printf("scheduler cancel offload collection=%s reason=activity_resumed", collection)
		log.Println("activity resumed, reverting to HOT:", collection)
		s.store.Set(collection, state.Hot)
		return
	}

	log.Println("scheduler offloading after drain:", collection)
	if err := s.lifecycleMgr.Offload(collection); err != nil {
		log.Println("offload failed:", collection, err)
		// fallback: revert state
		s.store.Set(collection, state.Hot)
	}
}
