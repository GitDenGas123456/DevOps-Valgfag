package metrics

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var RequestCount = promauto.NewCounter(prometheus.CounterOpts{
	Name: "whoknows_requests_total",
	Help: "Total number of API requests",
})

var SearchTotal = promauto.NewCounter(prometheus.CounterOpts{
	Name: "app_search_total",
	Help: "Total number of search requests",
})

var SearchWithResult = promauto.NewCounter(prometheus.CounterOpts{
	Name: "app_search_with_result_total",
	Help: "Number of search requests that returned at least one result",
})

var SearchLatency = promauto.NewHistogram(prometheus.HistogramOpts{
	Name: "app_search_duration_seconds",
	Help: "Search handler latency in seconds",
})

// HTTPRequestsTotal tracks all HTTP responses split by path template and status code.
var HTTPRequestsTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "app_http_requests_total",
		Help: "Total HTTP requests by path and status code",
	},
	[]string{"path", "code"},
)

// RequestMetricsMiddleware records status code and path for each request.
func RequestMetricsMiddleware() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)

			path := r.URL.Path
			if route := mux.CurrentRoute(r); route != nil {
				if tmpl, err := route.GetPathTemplate(); err == nil {
					path = tmpl
				}
			}

			HTTPRequestsTotal.WithLabelValues(path, strconv.Itoa(rec.status)).Inc()
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}
