package cognitive

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/moasq/go-b2b-starter/internal/platform/authcontext"
	billingServices "github.com/moasq/go-b2b-starter/internal/modules/billing/app/services"
)

// AiCreditGuard blocks AI routes for organizations whose period credits are
// exhausted (CreditsMax > 0 and no remaining credits). An unset allowance
// (CreditsMax = 0) never blocks.
type AiCreditGuard struct {
	billingService billingServices.BillingService
}

// NewAiCreditGuard creates an AI credit guard backed by the billing service.
func NewAiCreditGuard(billingService billingServices.BillingService) *AiCreditGuard {
	return &AiCreditGuard{billingService: billingService}
}

// RequireCredits returns middleware that aborts with HTTP 402 when the
// organization's period AI credits are exhausted. Must run after org_context.
func (g *AiCreditGuard) RequireCredits() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		orgID := authcontext.GetOrganizationID(c)
		if orgID == 0 {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "configuration_error",
				"mensaje": "Organization context required - ensure org_context middleware is applied",
			})
			c.Abort()
			return
		}

		status, err := g.billingService.GetAiUsageStatus(c.Request.Context(), orgID)
		if err != nil {
			// Fail-open on ledger errors: never block AI on infrastructure issues.
			c.Next()
			return
		}

		if status.CreditsMax > 0 && status.CreditsRemaining <= 0 {
			c.JSON(http.StatusPaymentRequired, gin.H{
				"error":   "ai_credits_exhausted",
				"mensaje": "Has agotado tus créditos de IA para este periodo. Actualiza tu plan o espera a que se renueven.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
