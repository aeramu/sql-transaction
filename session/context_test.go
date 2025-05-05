package session

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithTx(t *testing.T) {
	// Create a test value
	testTx := "test_transaction"

	// Create context with transaction
	ctx := context.Background()
	txCtx := WithTx(ctx, testTx)

	// Verify transaction was stored in context
	assert.Equal(t, testTx, txCtx.Value(txKey{}))
}

func TestGetTx(t *testing.T) {
	// Test with no transaction
	ctx := context.Background()
	tx := GetTx(ctx)
	assert.Nil(t, tx)

	// Test with transaction
	testTx := "test_transaction"
	txCtx := WithTx(ctx, testTx)
	tx = GetTx(txCtx)
	assert.Equal(t, testTx, tx)
}
