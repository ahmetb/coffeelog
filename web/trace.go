package main

import (
	"fmt"
	"net/http"
	"time"

	"cloud.google.com/go/trace"
	"cloud.google.com/go/trace/traceutil"
	"github.com/sirupsen/logrus"
)

type proxyResponseWriter struct {
	w      http.ResponseWriter
	code   int
	length int
}

func (p *proxyResponseWriter) Header() http.Header { return p.w.Header() }

func (p *proxyResponseWriter) WriteHeader(code int) { p.code = code; p.w.WriteHeader(code) }

func (p *proxyResponseWriter) Write(b []byte) (int, error) {
	n, err := p.w.Write(b)
	p.length += n
	return n, err
}

// traceHandler wraps the HTTP handler with tracing that automatically finishes
// the span. It adds additional fields to the trace span about the response.
func (s *server) traceHandler(h func(http.ResponseWriter, *http.Request)) http.Handler {
	return traceutil.HTTPHandler(s.tc, func(w http.ResponseWriter, r *http.Request) {
		ww := &proxyResponseWriter{w: w}
		span := trace.FromContext(r.Context())
		defer func() {
			code := ww.code
			if code == 0 {
				code = http.StatusOK
			}
			span.SetLabel("http/returned_status_code", fmt.Sprint(code))
			span.SetLabel("http/returned_content_length", fmt.Sprint(ww.length))
			span.Finish()
		}()
		h(ww, r)
	})
}

// logHandler wraps the HTTP handler with structured logging.
func logHandler(h func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		e := log.WithFields(logrus.Fields{
			"method": r.Method,
			"path":   r.URL.Path,
		})
		e.Debug("request accepted")
		start := time.Now()
		defer func() {
			e.WithFields(logrus.Fields{
				"elapsed": time.Now().Sub(start).String(),
			}).Debug("request completed")
		}()
		h(w, r)
	}
}
