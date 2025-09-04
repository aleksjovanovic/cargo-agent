package middleware

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/aleksjovanovic/cargo-agent/internal/ctxmeta"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// Total number of HTTP requests, labeled by method, route pattern and status.
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "cargo_agent",
			Subsystem: "http",
			Name:      "requests_total",
			Help:      "Total number of HTTP requests.",
		},
		[]string{"method", "route", "status"},
	)

	// Latency histogram for HTTP requests in seconds.
	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "cargo_agent",
			Subsystem: "http",
			Name:      "request_duration_seconds",
			Help:      "HTTP request latency in seconds.",
			Buckets:   prometheus.DefBuckets, // defaults: [0.005 ... 10]
		},
		[]string{"method", "route", "status"},
	)

	// Number of in-flight requests at any point in time.
	httpInFlight = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "cargo_agent",
			Subsystem: "http",
			Name:      "in_flight_requests",
			Help:      "Current number of in-flight HTTP requests.",
		},
	)
)

func init() {
	// Safe to register once per process; importing this package multiple times still
	// runs init once per binary.
	prometheus.MustRegister(httpRequestsTotal, httpRequestDuration, httpInFlight)
}

// statusRecorder wraps ResponseWriter to capture status code and bytes written.
// It also forwards optional interfaces (Flusher, Hijacker, Pusher, ReaderFrom)
// when the underlying writer supports them, so streaming and HTTP/2 features work.
type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *statusRecorder) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusRecorder) Write(b []byte) (int, error) {
	// If Write is called without an explicit WriteHeader, default to 200.
	if w.status == 0 {
		w.status = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(b)
	w.bytes += n
	return n, err
}

// Support http.Flusher if the underlying writer has it.
func (w *statusRecorder) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Support http.Hijacker if available (used by websockets, raw TCP).
func (w *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := w.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

// Support http.Pusher (HTTP/2 server push) if available.
func (w *statusRecorder) Push(target string, opts *http.PushOptions) error {
	if p, ok := w.ResponseWriter.(http.Pusher); ok {
		return p.Push(target, opts)
	}
	return http.ErrNotSupported
}

// Support io.ReaderFrom to enable optimized io.Copy into the writer when possible.
func (w *statusRecorder) ReadFrom(r io.Reader) (int64, error) {
	if rf, ok := w.ResponseWriter.(io.ReaderFrom); ok {
		n, err := rf.ReadFrom(r)
		// If WriteHeader wasn't called yet, assume 200 on first write path.
		if w.status == 0 && n > 0 {
			w.status = http.StatusOK
		}
		if n > 0 {
			w.bytes += int(n) // best-effort; fine for metrics
		}
		return n, err
	}
	// Fallback: generic copy through Write
	buf := make([]byte, 32*1024)
	var total int64
	for {
		nr, er := r.Read(buf)
		if nr > 0 {
			nw, ew := w.Write(buf[:nr])
			total += int64(nw)
			if ew != nil {
				return total, ew
			}
			if nw != nr {
				return total, io.ErrShortWrite
			}
		}
		if er != nil {
			if er == io.EOF {
				break
			}
			return total, er
		}
	}
	return total, nil
}

// Metrics is an HTTP middleware that records request count, status and duration.
// The "route" label should be LOW cardinality. Prefer a normalized route pattern,
// e.g. "/cargo-agent/v1/cargo-offers/{id}", not the concrete path with IDs.
//
// How we determine the route label, in order:
//  1. ctxmeta.RoutePattern (if another middleware set it in the context)
//  2. r.Pattern (Go 1.23+ field on *http.Request populated by ServeMux)
//  3. "unknown" as the final fallback
func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpInFlight.Inc()
		defer httpInFlight.Dec()

		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w}

		// Call the next handler in the chain.
		next.ServeHTTP(rec, r)

		// Derive labels.
		method := r.Method
		status := rec.status
		if status == 0 {
			// If nothing set the status explicitly, assume 200.
			status = http.StatusOK
		}

		route := "unknown"

		// Prefer a normalized route pattern injected into context by a router/middleware.
		if p, ok := ctxmeta.RoutePattern(r.Context()); ok && p != "" {
			route = p
		} else if r.Pattern != "" {
			// Go 1.23+: Request.Pattern is a string field set by ServeMux.
			route = r.Pattern
		}

		labels := prometheus.Labels{
			"method": method,
			"route":  route,
			"status": strconv.Itoa(status),
		}

		// Update metrics.
		httpRequestsTotal.With(labels).Inc()
		httpRequestDuration.With(labels).Observe(time.Since(start).Seconds())
	})
}
