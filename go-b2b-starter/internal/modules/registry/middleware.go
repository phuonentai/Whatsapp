package registry

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/moasq/go-b2b-starter/internal/platform/features"
)

// Require returns middleware that gates a route group on a module being
// enabled for the requesting organization. It runs after
// EntitlementMiddleware, mirroring features.Require ordering.
func Require(moduleKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		entitlement := features.GetEntitlement(c)
		if entitlement == nil {
			c.JSON(http.StatusForbidden, gin.H{
				"error":  "module_unavailable",
				"module": moduleKey,
			})
			c.Abort()
			return
		}

		if state, ok := entitlement.Modules[moduleKey]; !ok || !state.Enabled {
			c.JSON(http.StatusForbidden, gin.H{
				"error":  "module_disabled",
				"module": moduleKey,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
