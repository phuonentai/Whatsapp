package cognitive

import (
	"github.com/gin-gonic/gin"

	"github.com/moasq/go-b2b-starter/internal/modules/auth"
	"github.com/moasq/go-b2b-starter/internal/platform/authcontext"
	"github.com/moasq/go-b2b-starter/internal/platform/features"
	serverDomain "github.com/moasq/go-b2b-starter/internal/platform/server/domain"
)

// FeatureAIAssistant gates AI routes (derived from subscription metadata).
const FeatureAIAssistant = "ai_assistant"

type Routes struct {
	handler         *Handler
	featureProvider features.FeatureProvider
	creditGuard     *AiCreditGuard
}

func NewRoutes(
	handler *Handler,
	featureProvider features.FeatureProvider,
	creditGuard *AiCreditGuard,
) *Routes {
	return &Routes{
		handler:         handler,
		featureProvider: featureProvider,
		creditGuard:     creditGuard,
	}
}

func (r *Routes) RegisterRoutes(router *gin.RouterGroup, resolver serverDomain.MiddlewareResolver) {
	cognitiveGroup := router.Group("/example_cognitive")
	cognitiveGroup.Use(
		resolver.Get("auth"),
		resolver.Get("org_context"),
		features.EntitlementMiddleware(r.featureProvider, authcontext.GetRequestContext),
		resolver.Get("subscription"),
	)
	{
		// Chat endpoint — gated by the ai_assistant feature flag, then by the
		// AI credit guard (402 when period credits are exhausted).
		cognitiveGroup.POST("/chat",
			features.Require(FeatureAIAssistant),
			r.creditGuard.RequireCredits(),
			auth.RequirePermissionFunc("resource", "create"),
			r.handler.Chat)

		// Chat sessions
		sessionsGroup := cognitiveGroup.Group("/sessions")
		{
			sessionsGroup.GET("",
				auth.RequirePermissionFunc("resource", "view"),
				r.handler.ListSessions)

			sessionsGroup.GET("/:id/messages",
				auth.RequirePermissionFunc("resource", "view"),
				r.handler.GetSessionHistory)
		}
	}
}

// Routes returns a RouteRegistrar function compatible with the server interface
func (r *Routes) Routes(router *gin.RouterGroup, resolver serverDomain.MiddlewareResolver) {
	r.RegisterRoutes(router, resolver)
}
