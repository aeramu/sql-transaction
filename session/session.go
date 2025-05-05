package session

import (
	"context"
	"database/sql"
	"fmt"
)

type Session interface {
	WithTransaction(ctx context.Context, f func(ctx context.Context) error) error
}

func NewSession(db *sql.DB) Session {
	return &session{db: db}
}

type session struct {
	db *sql.DB
}

// WithTransaction runs the function f in a transaction.
// If a transaction is already in progress, it will be used instead of starting a new one.
// If a transaction is not in progress, it will start a new one.
// If the function f returns an error, the transaction will be rolled back.
// If the function f returns nil, the transaction will be committed.
func (s *session) WithTransaction(ctx context.Context, f func(ctx context.Context) error) error {
	tx, ok := GetTx(ctx).(*sql.Tx)
	if ok && tx != nil {
		return f(ctx)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	ctx = WithTx(ctx, tx)

	defer func() {
		if p := recover(); p != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				fmt.Printf("rollback error during panic: %v\n", rbErr)
			}
			panic(p)
		}
	}()

	err = f(ctx)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("rollback error: %w (original error: %v)", rbErr, err)
		}
		return fmt.Errorf("transaction failed: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit error: %w", err)
	}
	return nil
}
