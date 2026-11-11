package http

import (
	"bytes"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// IdempotencyCache caches completed responses for given idempotency keys.
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
	// Background reaper
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

// IdempotencyMiddleware ensures duplicate requests with the same X-Idempotency-Key
// return the cached response without re-executing stateful business logic.
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

// CORSMiddleware sets permissive CORS for local LAN hardware and frontends.
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Idempotency-Key, X-Requested-With")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// AuthMiddleware optionally checks bearer token if expected.
func AuthMiddleware(expectedToken string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if expectedToken == "" {
			c.Next()
			return
		}

		token := c.GetHeader("Authorization")
		if token != "Bearer "+expectedToken && token != expectedToken {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "unauthorized: invalid or missing bearer token",
			})
			return
		}

		c.Next()
	}
}
