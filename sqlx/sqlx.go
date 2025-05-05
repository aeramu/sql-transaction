package transaction

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"

	"github.com/aeramu/sql-transaction/session"
)

type Executor interface {
	sqlx.Ext
	sqlx.ExtContext
	sqlx.Preparer
	sqlx.PreparerContext
}

func New(db *sqlx.DB) session.DBWrapper[Executor] {
	return session.NewDBWrapper(&DB{
		db: db,
	})
}

type DB struct {
	db *sqlx.DB
}

func (s *DB) ConvertTx(ctx context.Context, tx *sql.Tx) Executor {
	return &sqlx.Tx{
		Tx: tx,
	}
}

func (s *DB) GetDB(ctx context.Context) Executor {
	return s.db
}
