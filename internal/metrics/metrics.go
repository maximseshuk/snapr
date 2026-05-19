package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	TotalAttempts = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "snapr",
			Name:      "backup_attempts_total",
			Help:      "Total number of backup attempts.",
		},
		[]string{"job", "status"},
	)

	DurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "snapr",
			Name:      "backup_duration_seconds",
			Help:      "Backup execution duration in seconds.",
			Buckets:   prometheus.ExponentialBuckets(1, 2, 15),
		},
		[]string{"job"},
	)

	FileSizeBytes = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "snapr",
			Name:      "backup_file_size_bytes",
			Help:      "Size of the most recent backup archive in bytes.",
		},
		[]string{"job"},
	)

	LastTimestamp = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "snapr",
			Name:      "backup_last_timestamp_seconds",
			Help:      "Unix timestamp of the most recent backup attempt.",
		},
		[]string{"job", "status"},
	)
)

func ObserveSuccess(jobName string, durationSec float64, archiveSize int64) {
	TotalAttempts.WithLabelValues(jobName, "success").Inc()
	DurationSeconds.WithLabelValues(jobName).Observe(durationSec)
	LastTimestamp.WithLabelValues(jobName, "success").SetToCurrentTime()
	if archiveSize > 0 {
		FileSizeBytes.WithLabelValues(jobName).Set(float64(archiveSize))
	}
}

func ObserveFailure(jobName string, durationSec float64) {
	TotalAttempts.WithLabelValues(jobName, "failure").Inc()
	DurationSeconds.WithLabelValues(jobName).Observe(durationSec)
	LastTimestamp.WithLabelValues(jobName, "failure").SetToCurrentTime()
}
