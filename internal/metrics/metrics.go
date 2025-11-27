package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var RequestCount = promauto.NewCounter(prometheus.CounterOpts{
    Name: "whoknows_requests_total",
    Help: "Total number of API requests",
})
