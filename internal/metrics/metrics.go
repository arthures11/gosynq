package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics struct {
	JobsProcessed    prometheus.Counter
	JobsFailed       prometheus.Counter
	JobsSucceeded    prometheus.Counter
	JobsRetried      prometheus.Counter
	ActiveWorkers    prometheus.Gauge
	QueueLength      prometheus.Gauge
	ProcessingTime   prometheus.Histogram
	EventChannelSize prometheus.Gauge
}

func NewMetrics() *Metrics {
	return &Metrics{
		JobsProcessed: promauto.NewCounter(prometheus.CounterOpts{
			Name: "gosynq_jobs_processed_total",
			Help: "Total number of jobs processed",
		}),
		JobsFailed: promauto.NewCounter(prometheus.CounterOpts{
			Name: "gosynq_jobs_failed_total",
			Help: "Total number of jobs that failed",
		}),
		JobsSucceeded: promauto.NewCounter(prometheus.CounterOpts{
			Name: "gosynq_jobs_succeeded_total",
			Help: "Total number of jobs that succeeded",
		}),
		JobsRetried: promauto.NewCounter(prometheus.CounterOpts{
			Name: "gosynq_jobs_retried_total",
			Help: "Total number of jobs that were retried",
		}),
		ActiveWorkers: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "gosynq_active_workers",
			Help: "Number of active workers",
		}),
		QueueLength: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "gosynq_queue_length",
			Help: "Current queue length",
		}),
		ProcessingTime: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "gosynq_job_processing_time_seconds",
			Help:    "Time taken to process jobs",
			Buckets: prometheus.DefBuckets,
		}),
		EventChannelSize: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "gosynq_event_channel_size",
			Help: "Current size of the event channel",
		}),
	}
}

func (m *Metrics) IncJobsProcessed() {
	m.JobsProcessed.Inc()
}

func (m *Metrics) IncJobsFailed() {
	m.JobsFailed.Inc()
}

func (m *Metrics) IncJobsSucceeded() {
	m.JobsSucceeded.Inc()
}

func (m *Metrics) IncJobsRetried() {
	m.JobsRetried.Inc()
}

func (m *Metrics) SetActiveWorkers(count int) {
	m.ActiveWorkers.Set(float64(count))
}

func (m *Metrics) SetQueueLength(length int) {
	m.QueueLength.Set(float64(length))
}

func (m *Metrics) ObserveProcessingTime(duration float64) {
	m.ProcessingTime.Observe(duration)
}

func (m *Metrics) SetEventChannelSize(size int) {
	m.EventChannelSize.Set(float64(size))
}
