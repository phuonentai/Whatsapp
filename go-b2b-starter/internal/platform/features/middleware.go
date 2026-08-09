package features

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/moasq/go-b2b-starter/internal/modules/auth"
)

func Require(featureName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		entitlement := GetEntitlement(c)
		if entitlement == nil {
			c.JSON(http.StatusForbidden, gin.H{
				"error":          "funcionalidad_no_disponible",
				"funcionalidad":  featureName,
				"mensaje":        "No se pudo verificar la disponibilidad de esta funcionalidad.",
			})
			c.Abort()
			return
		}

		if entitlement.IsReadOnly && c.Request.Method != http.MethodGet {
			c.JSON(http.StatusForbidden, gin.H{
				"error":      "solo_lectura",
				"mensaje":    "Tu suscripción está en modo de solo lectura. Reactívala para hacer cambios.",
				"plan":       entitlement.PlanName,
			})
			c.Abort()
			return
		}

		if enabled, ok := entitlement.Features[featureName]; !ok || !enabled {
			c.JSON(http.StatusForbidden, gin.H{
				"error":          "funcionalidad_no_disponible",
				"funcionalidad":  featureName,
				"mensaje":        "Esta funcionalidad no está disponible en tu plan actual. Actualiza tu plan para acceder.",
				"plan":           entitlement.PlanName,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

func RequireActiveSubscription() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		entitlement := GetEntitlement(c)
		if entitlement == nil || (len(entitlement.Features) == 0 && !entitlement.IsGracePeriod) {
			c.JSON(http.StatusPaymentRequired, gin.H{
				"error":   "suscripcion_requerida",
				"mensaje": "Se requiere una suscripción activa para acceder al CRM.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

type RequestContextFunc func(c *gin.Context) *auth.RequestContext

func EntitlementMiddleware(provider FeatureProvider, getOrgCtx RequestContextFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		reqCtx := getOrgCtx(c)
		if reqCtx == nil {
			c.Next()
			return
		}

		entitlement, err := provider.GetEntitlement(c.Request.Context(), reqCtx.OrganizationID)
		if err != nil || entitlement == nil {
			entitlement = &Entitlement{
				Features: make(map[string]bool),
				Quotas:   make(map[string]int32),
				Usage:    make(map[string]int32),
				Modules:  make(map[string]ModuleState),
			}
		}

		SetEntitlement(c, entitlement)
		c.Next()
	}
}
