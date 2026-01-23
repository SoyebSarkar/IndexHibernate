package metrics

import (
	"github.com/SoyebSarkar/Hiberstack/internal/state"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// -------- Counters --------

	ReloadTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "hiberstack_reload_total",
		Help: "Total number of collection reloads",
	})

	OffloadTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "hiberstack_offload_total",
		Help: "Total number of collection offloads",
	})

	WriteBlockedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "hiberstack_write_blocked_total",
		Help: "Total number of write requests blocked during draining",
	})

	// -------- Gauges --------

	CollectionsHot = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "hiberstack_collections_hot",
		Help: "Number of HOT collections",
	})

	CollectionsCold = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "hiberstack_collections_cold",
		Help: "Number of COLD collections",
	})

	CollectionsDraining = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "hiberstack_collections_draining",
		Help: "Number of DRAINING collections",
	})

	CollectionsLoading = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "hiberstack_collections_loading",
		Help: "Number of LOADING collections",
	})

	// -------- Histograms --------

	ReloadDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "hiberstack_reload_duration_seconds",
		Help:    "Time taken to reload a collection",
		Buckets: prometheus.DefBuckets,
	})

	BlockingReloadWait = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "hiberstack_blocking_reload_wait_seconds",
		Help:    "Time a request waits for a blocking reload to finish",
		Buckets: prometheus.DefBuckets,
	})
)

func UpdateStateGauges(store *state.Store) {
	counts := store.CountByState()

	CollectionsHot.Set(float64(counts[state.Hot]))
	CollectionsCold.Set(float64(counts[state.Cold]))
	CollectionsDraining.Set(float64(counts[state.Draining]))
	CollectionsLoading.Set(float64(counts[state.Loading]))
}
