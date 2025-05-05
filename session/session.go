package session

import (
	"context"
	"database/sql"
	"fmt"
)

// Session provides transaction management capabilities for database operations
type Session interface {
	WithTransaction(ctx context.Context, f func(ctx context.Context) error, opts ...*sql.TxOptions) error
}

// Executor defines the common database operations that can be performed by both *sql.DB and *sql.Tx
type Executor interface {
	Exec(query string, args ...any) (sql.Result, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	Prepare(query string) (*sql.Stmt, error)
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type session struct {
	db *sql.DB
}

func NewSession(db *sql.DB) Session {
	return &session{db: db}
}

func getTx(ctx context.Context) (*sql.Tx, bool) {
	val := GetTx(ctx)
	tx, ok := val.(*sql.Tx)
	if !ok || tx == nil {
		return nil, false
	}
	return tx, true
}

func (s *session) WithTransaction(ctx context.Context, f func(ctx context.Context) error, opts ...*sql.TxOptions) error {
	tx, txExist := getTx(ctx)
	if txExist {
		return f(ctx)
	}

	var err error
	var txOpt *sql.TxOptions
	if len(opts) > 0 {
		txOpt = opts[0]
	}
	tx, err = s.db.BeginTx(ctx, txOpt)
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

// GetDB returns the appropriate executor (either transaction or database connection)
func (s *session) GetDB(ctx context.Context) Executor {
	tx, exist := getTx(ctx)
	if !exist {
		return s.db
	}
	return tx
}
