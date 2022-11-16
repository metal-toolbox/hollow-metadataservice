package middleware

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// MetricMetadataCacheHit total number of metadata requests not requiring external lookups
	MetricMetadataCacheHit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "metadata_cache_hit_total",
		Help: "Number of metadata requests that were immediately found in the db.",
	})

	// MetricMetadataCacheMiss total number of metadata requests that required external lookups
	MetricMetadataCacheMiss = promauto.NewCounter(prometheus.CounterOpts{
		Name: "metadata_cache_miss_total",
		Help: "Number of metadata requests not found in the db that needed to be sent to the lookup service.",
	})

	// MetricUserdataCacheHit total number of userdata requests not requiring external lookups
	MetricUserdataCacheHit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "metadata_userdata_cache_hit_total",
		Help: "Number of userdata requests that were immediately found in the db.",
	})

	// MetricUserdataCacheMiss total number of requests that required external lookups
	MetricUserdataCacheMiss = promauto.NewCounter(prometheus.CounterOpts{
		Name: "metadata_userdata_cache_miss_total",
		Help: "Number of userdata requests not found in the db that needed to be sent to the lookup service.",
	})

	// MetricMetadataLookupRequestCount total number of metadata requests sent to the external lookup service
	MetricMetadataLookupRequestCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "metadata_lookup_request_total",
		Help: "Number of metadata lookup requests.",
	})

	// MetricUserdataLookupRequestCount total number of userdata requests sent to the external lookup service
	MetricUserdataLookupRequestCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "metadata_userdata_lookup_request_total",
		Help: "Number of userdata lookup requests.",
	})

	// MetricMetadataInsertsCount total number of metadata inserts (which originate from the API)
	MetricMetadataInsertsCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "metadata_inserts_total",
		Help: "Number of metadata inserts (which originate from the API).",
	})

	// MetricUserdataInsertsCount total number of userdata inserts (which originate from the API)
	MetricUserdataInsertsCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "metadata_userdata_inserts_total",
		Help: "Number of userdata inserts (which originate from the API).",
	})

	// MetricDeletionsCount total number of metadata deletions (which originate from the API)
	MetricDeletionsCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "metadata_deletions_total",
		Help: "Number of metadata deletions (which originate from the API).",
	})

	// MetricLookupErrors total number of errors produced during external lookup requests
	MetricLookupErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "metadata_lookup_error_total",
		Help: "Number of errors produced during metadata lookups.",
	})

	// MetricMetadataStoreErrors total number of errors produced during saving/updating metadata to the db
	MetricMetadataStoreErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "metadata_store_error_total",
		Help: "Number of errors produced while saving or updating metadata to the database.",
	})

	// MetricUserdataLookupErrors total number of errors produced during external userdata lookup requests
	MetricUserdataLookupErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "metadata_userdata_lookup_error_total",
		Help: "Number of errors produced during metadata lookups.",
	})

	// MetricUserdataStoreErrors total number of errors produced during saving/updating userdata to the db
	MetricUserdataStoreErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "metadata_userdata_store_error_total",
		Help: "Number of errors produced while saving or updating userdata to the database.",
	})
)
