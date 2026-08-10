package domain

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const probeTimeout = 2 * time.Second

func (s *HTTPServer) setupHealthCheck() {
	// Liveness: the process is up. Always 200; orchestrators must not kill a
	// process that is still alive but waiting on dependencies.
	liveness := func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "OK"})
	}

	// Readiness: dependencies (Postgres, Redis) are reachable. 503 with
	// dependency detail when a probe fails.
	readiness := func(c *gin.Context) {
		failures := make([]string, 0, 2)

		ctx, cancel := context.WithTimeout(c.Request.Context(), probeTimeout)
		defer cancel()

		if s.db != nil {
			if err := s.db.Ping(ctx); err != nil {
				failures = append(failures, "postgres")
			}
		}
		if s.redisClient != nil {
			if err := s.redisClient.Ping(ctx); err != nil {
				failures = append(failures, "redis")
			}
		}

		if len(failures) > 0 {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":  "unavailable",
				"failed":  failures,
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	}

	// /health stays for backwards compatibility (CI + Caddy) and now reflects
	// dependency readiness in non-prod only; production uses /readyz.
	s.router.GET("/health", func(c *gin.Context) {
		if s.config.IsProd() {
			readiness(c)
			return
		}
		readiness(c)
	})
	s.router.GET("/api/health", readiness)
	s.router.GET("/healthz", liveness)
	s.router.GET("/readyz", readiness)
	s.logger.Info("Health check endpoints set up: /health, /api/health, /healthz, /readyz")
}

func (s *HTTPServer) setupRootEndpoint() {
	s.router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service":   "B2B SaaS Starter API",
			"version":   "1.0.0",
			"status":    "running",
			"health":    "/api/health",
			"docs":      "/api/docs",
			"timestamp": time.Now().UTC(),
		})
	})
	s.logger.Info("Root endpoint set up at /")
}
