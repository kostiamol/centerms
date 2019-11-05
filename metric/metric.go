package metric

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/kostiamol/centerms/log"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metric struct {
	serviceTiming *prometheus.SummaryVec
	errorCounter  *prometheus.CounterVec
}

func New(appID string) *Metric {
	r := strings.NewReplacer(
		"-", "_",
		" ", "_")
	serviceName := r.Replace(appID)

	m := &Metric{
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

func (m *Metric) ErrorCounter(label string) {
	m.errorCounter.
		WithLabelValues(label).
		Inc()
}

func (m *Metric) Timing(start time.Time, label string) {
	m.serviceTiming.
		WithLabelValues(label).
		Observe(time.Since(start).Seconds())
}

func (m *Metric) TimeTracker(next http.HandlerFunc, label string, l log.Logger) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		start := time.Now()
		next(response, request)
		m.Timing(start, label)
	}
}

func (m *Metric) RouterHandlerHTTP() http.HandlerFunc {
	return m.stdToHTTPRouterMiddleware(m.handlerHTTP())
}

func (m *Metric) handlerHTTP() http.Handler {
	return promhttp.Handler()
}

func (m *Metric) stdToHTTPRouterMiddleware(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	}
}
