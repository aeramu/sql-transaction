package session

import "context"

type txKey struct{}

// WithTx returns a new context with the given transaction value
func WithTx(ctx context.Context, tx any) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

// GetTx retrieves a transaction from the context if it exists
func GetTx(ctx context.Context) any {
	return ctx.Value(txKey{})
}
