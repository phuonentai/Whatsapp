package billing

import (
	"github.com/gin-gonic/gin"

	"github.com/moasq/go-b2b-starter/internal/modules/auth"
	serverDomain "github.com/moasq/go-b2b-starter/internal/platform/server/domain"
)

func (h *Handler) Routes(router *gin.RouterGroup, resolver serverDomain.MiddlewareResolver) {
	subscriptions := router.Group("/subscriptions")
	subscriptions.Use(
		resolver.Get("auth"),
		resolver.Get("org_context"),
	)
	{
		subscriptions.GET("/status",
			auth.RequirePermissionFunc("resource", "view"),
			h.GetBillingStatus)
	}

	router.POST("/subscriptions/verify-payment",
		resolver.Get("auth"),
		h.VerifyPayment)
}
