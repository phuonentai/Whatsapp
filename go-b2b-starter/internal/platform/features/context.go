package features

import (
	"context"

	"github.com/gin-gonic/gin"
)

type contextKey string

const entitlementKey contextKey = "entitlement"

func SetEntitlement(c *gin.Context, entitlement *Entitlement) {
	c.Set(string(entitlementKey), entitlement)
}

func GetEntitlement(c *gin.Context) *Entitlement {
	if val, exists := c.Get(string(entitlementKey)); exists {
		if e, ok := val.(*Entitlement); ok {
			return e
		}
	}
	return nil
}

func WithEntitlement(ctx context.Context, entitlement *Entitlement) context.Context {
	return context.WithValue(ctx, entitlementKey, entitlement)
}

func FromContext(ctx context.Context) *Entitlement {
	if val := ctx.Value(entitlementKey); val != nil {
		if e, ok := val.(*Entitlement); ok {
			return e
		}
	}
	return nil
}
