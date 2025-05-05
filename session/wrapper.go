package session

import (
	"context"
	"database/sql"
)

type Database[T any] interface {
	GetDB(ctx context.Context) T
	ConvertTx(ctx context.Context, tx *sql.Tx) T
}

type DBWrapper[T any] interface {
	GetDB(ctx context.Context) T
}

func NewDBWrapper[T any](db Database[T]) DBWrapper[T] {
	return &wrapper[T]{
		db: db,
	}
}

type wrapper[T any] struct {
	db Database[T]
}

func (w *wrapper[T]) GetDB(ctx context.Context) T {
	val := GetTx(ctx)
	tx, ok := val.(*sql.Tx)
	if !ok || tx == nil {
		return w.db.GetDB(ctx)
	}
	return w.db.ConvertTx(ctx, tx)
}
