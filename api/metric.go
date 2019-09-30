package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/kostiamol/centerms/log"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type metric struct {
	serviceTiming *prometheus.SummaryVec
	errorCounter  *prometheus.CounterVec
}

func newMetric(appID string) *metric {
	r := strings.NewReplacer(
		"-", "_",
		" ", "_")
	serviceName := r.Replace(appID)

	m := &metric{
		serviceTiming: prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Name: "service_timing",
				Help: fmt.Sprintf("%s timing", serviceName),
			},
			[]string{fmt.Sprintf("%s_service", serviceName)},
		),
		errorCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "error_counter",
				Help: fmt.Sprintf("%s error counter", serviceName),
			},
			[]string{fmt.Sprintf("%s_error", serviceName)},
		),
	}

	prometheus.MustRegister(m.serviceTiming)
	prometheus.MustRegister(m.errorCounter)

	return m
}

func (m *metric) ErrorCounter(label string) {
	m.errorCounter.
		WithLabelValues(label).
		Inc()
}

func (m *metric) timing(start time.Time, label string) {
	m.serviceTiming.
		WithLabelValues(label).
		Observe(time.Since(start).Seconds())
}

func (m *metric) timeTracker(next http.HandlerFunc, label string, l log.Logger) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		start := time.Now()
		next(response, request)
		m.timing(start, label)
	}
}

func (m *metric) httpHandler() http.Handler {
	return promhttp.Handler()
}

func (m *metric) httpRouterHandler() http.HandlerFunc {
	return m.stdToHTTPRouterMiddleware(m.httpHandler())
}

func (m *metric) stdToHTTPRouterMiddleware(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	}
}
