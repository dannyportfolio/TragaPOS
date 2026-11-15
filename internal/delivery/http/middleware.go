package http

import (
	"bytes"
	"crypto/subtle"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"

	"pos-engine/internal/pkg/uid"
)

type responseCacheItem struct {
	status int
	body   []byte
	header http.Header
	exp    time.Time
}

type IdempotencyManager struct {
	mu    sync.RWMutex
	store map[string]responseCacheItem
}

func NewIdempotencyManager() *IdempotencyManager {
	m := &IdempotencyManager{
		store: make(map[string]responseCacheItem),
	}
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			m.mu.Lock()
			now := time.Now()
			for k, v := range m.store {
				if now.After(v.exp) {
					delete(m.store, k)
				}
			}
			m.mu.Unlock()
		}
	}()
	return m
}

type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (m *IdempotencyManager) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodPost && c.Request.Method != http.MethodPut {
			c.Next()
			return
		}

		key := c.GetHeader("X-Idempotency-Key")
		if key == "" {
			c.Next()
			return
		}

		m.mu.RLock()
		cached, exists := m.store[key]
		m.mu.RUnlock()

		if exists && time.Now().Before(cached.exp) {
			for k, vals := range cached.header {
				for _, val := range vals {
					c.Header(k, val)
				}
			}
			c.Header("X-Cache-Lookup", "HIT-IDEMPOTENCY")
			c.Data(cached.status, c.Writer.Header().Get("Content-Type"), cached.body)
			c.Abort()
			return
		}

		blw := &bodyLogWriter{body: bytes.NewBuffer(nil), ResponseWriter: c.Writer}
		c.Writer = blw
		c.Next()

		if c.Writer.Status() >= 200 && c.Writer.Status() < 300 {
			m.mu.Lock()
			m.store[key] = responseCacheItem{
				status: c.Writer.Status(),
				body:   blw.body.Bytes(),
				header: c.Writer.Header().Clone(),
				exp:    time.Now().Add(10 * time.Minute),
			}
			m.mu.Unlock()
		}
	}
}

func CORSMiddleware(allowedOrigins []string) gin.HandlerFunc {
	allowAll := len(allowedOrigins) == 0
	allowedMap := make(map[string]bool, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		if origin == "*" {
			allowAll = true
			break
		}
		allowedMap[origin] = true
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if allowAll {
			c.Header("Access-Control-Allow-Origin", "*")
		} else if origin != "" && allowedMap[origin] {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Idempotency-Key, X-Requested-With")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func AuthMiddleware(expectedToken string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if expectedToken == "" {
			c.Next()
			return
		}

		token := strings.TrimSpace(c.GetHeader("Authorization"))
		if strings.HasPrefix(token, "Bearer ") {
			token = strings.TrimSpace(strings.TrimPrefix(token, "Bearer "))
		}
		if token == "" {
			token = strings.TrimSpace(c.Query("token"))
		}

		if subtle.ConstantTimeCompare([]byte(token), []byte(expectedToken)) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "unauthorized: invalid or missing bearer token",
			})
			return
		}

		c.Next()
	}
}

func StructuredLoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		reqID := c.GetHeader("X-Request-ID")
		if reqID == "" {
			reqID = uid.NewV7()
		}
		c.Header("X-Request-ID", reqID)

		c.Next()

		latencyMs := float64(time.Since(start).Microseconds()) / 1000.0
		slog.Info("http_request",
			"request_id", reqID,
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"latency_ms", latencyMs,
			"client_ip", c.ClientIP(),
		)
	}
}

type metricsRegistry struct {
	requestsTotal atomic.Uint64
	errors5xx     atomic.Uint64
	errors4xx     atomic.Uint64
	durationNanos atomic.Uint64
}

var appMetrics = &metricsRegistry{}

func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		elapsed := time.Since(start)

		appMetrics.requestsTotal.Add(1)
		appMetrics.durationNanos.Add(uint64(elapsed.Nanoseconds()))
		status := c.Writer.Status()
		if status >= 500 {
			appMetrics.errors5xx.Add(1)
		} else if status >= 400 {
			appMetrics.errors4xx.Add(1)
		}
	}
}

func MetricsHandler(c *gin.Context) {
	total := appMetrics.requestsTotal.Load()
	err4 := appMetrics.errors4xx.Load()
	err5 := appMetrics.errors5xx.Load()
	durSec := float64(appMetrics.durationNanos.Load()) / 1e9

	body := fmt.Sprintf(
		"# HELP pos_http_requests_total Total HTTP requests processed\n"+
			"# TYPE pos_http_requests_total counter\n"+
			"pos_http_requests_total %d\n"+
			"# HELP pos_http_errors_4xx_total Total 4xx client error responses\n"+
			"# TYPE pos_http_errors_4xx_total counter\n"+
			"pos_http_errors_4xx_total %d\n"+
			"# HELP pos_http_errors_5xx_total Total 5xx server error responses\n"+
			"# TYPE pos_http_errors_5xx_total counter\n"+
			"pos_http_errors_5xx_total %d\n"+
			"# HELP pos_http_request_duration_seconds_sum Total HTTP request duration in seconds\n"+
			"# TYPE pos_http_request_duration_seconds_sum counter\n"+
			"pos_http_request_duration_seconds_sum %.6f\n",
		total, err4, err5, durSec,
	)
	c.Data(http.StatusOK, "text/plain; version=0.0.4; charset=utf-8", []byte(body))
}
