// Package dbctx propagates a request-scoped database handle (transaction) via
// context.Context.
//
// conversation-row-scoping (Decisión 9 del design): cuando la capa RLS opt-in
// está activa, el middleware de scope abre la transacción del request, setea
// las session vars con `SET LOCAL` (nunca SET a nivel sesión sobre el pool) y
// pone la store transaccional en el contexto. Los repositorios que leen
// crm.conversations prefieren la store del contexto cuando existe; si no,
// caen a la store del pool (enforcement primario: query layer org-scoped).
package dbctx

import (
	"context"
)

type contextKey string

const storeKey contextKey = "request_store"

// WithStore attaches a request-scoped store (transaction) to the context.
func WithStore(ctx context.Context, store any) context.Context {
	return context.WithValue(ctx, storeKey, store)
}

// StoreFrom returns the request-scoped store if present, otherwise the
// fallback (pool) store.
func StoreFrom[T any](ctx context.Context, fallback T) T {
	if v, ok := ctx.Value(storeKey).(T); ok {
		return v
	}
	return fallback
}
