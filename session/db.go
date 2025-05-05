package session

import (
	"context"
	"database/sql"
)

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

func NewDB(db *sql.DB) DBWrapper[Executor] {
	return NewDBWrapper(&DB{sqlDB: db})
}

type DB struct {
	sqlDB *sql.DB
}

func (db *DB) GetDB(ctx context.Context) Executor {
	return db.sqlDB
}

func (db *DB) ConvertTx(tx *sql.Tx) Executor {
	return tx
}
