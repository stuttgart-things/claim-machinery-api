package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

// responseRecorder wraps http.ResponseWriter to capture status and size
type responseRecorder struct {
	http.ResponseWriter
	status int
	size   int
}

func (rw *responseRecorder) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseRecorder) Write(b []byte) (int, error) {
	if rw.status == 0 {
		rw.status = http.StatusOK
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.size += n
	return n, err
}

// context key type to avoid collisions
type ctxKey string

const ctxRequestIDKey ctxKey = "requestID"

func newRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// Fallback to timestamp-based ID
		return fmt.Sprintf("rid-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}

// requestIDMiddleware ensures a request ID is present and propagated
func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rid := r.Header.Get("X-Request-ID")
		if rid == "" {
			rid = newRequestID()
		}
		// set response header so clients can read it (and CORS will expose it)
		w.Header().Set("X-Request-ID", rid)

		ctx := context.WithValue(r.Context(), ctxRequestIDKey, rid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// loggingMiddleware logs all HTTP requests
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &responseRecorder{ResponseWriter: w}

		next.ServeHTTP(rec, r)

		duration := time.Since(start)
		remote := r.RemoteAddr
		ua := r.UserAgent()
		reqID, _ := r.Context().Value(ctxRequestIDKey).(string)

		if os.Getenv("LOG_FORMAT") == "json" {
			entry := map[string]interface{}{
				"ts":        time.Now().Format(time.RFC3339Nano),
				"method":    r.Method,
				"path":      r.RequestURI,
				"status":    rec.status,
				"bytes":     rec.size,
				"duration":  duration.String(),
				"remote":    remote,
				"ua":        ua,
				"requestId": reqID,
			}
			b, _ := json.Marshal(entry)
			log.Println(string(b))
			return
		}

		log.Printf("%s %s -> %d (%s) reqId=%s remote=%s ua=%q bytes=%d", r.Method, r.RequestURI, rec.status, duration, reqID, remote, ua, rec.size)
	})
}

// corsMiddleware adds CORS headers to allow cross-origin requests
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
		w.Header().Set("Access-Control-Expose-Headers", "X-Request-ID")

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// errorHandlerMiddleware handles panics and errors
func errorHandlerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// resolve request id from request or response header
				rid := r.Header.Get("X-Request-ID")
				if rid == "" {
					rid = w.Header().Get("X-Request-ID")
				}

				if os.Getenv("LOG_FORMAT") == "json" {
					entry := map[string]interface{}{
						"ts":        time.Now().Format(time.RFC3339Nano),
						"level":     "error",
						"event":     "panic",
						"error":     fmt.Sprintf("%v", err),
						"method":    r.Method,
						"path":      r.RequestURI,
						"remote":    r.RemoteAddr,
						"ua":        r.UserAgent(),
						"requestId": rid,
					}
					b, _ := json.Marshal(entry)
					log.Println(string(b))
				} else {
					log.Printf("❌ Panic recovered: %v reqId=%s method=%s path=%s", err, rid, r.Method, r.RequestURI)
				}

				w.Header().Set("Content-Type", "application/json")
				if rid != "" {
					w.Header().Set("X-Request-ID", rid)
				}
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, `{"error":"internal server error","requestId":"%s"}`, rid)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
