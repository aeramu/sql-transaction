package transaction

import (
	"context"
	"database/sql"

	"gorm.io/gorm"

	"github.com/aeramu/sql-transaction/session"
)

func NewDB(db *gorm.DB) session.DBWrapper[*gorm.DB] {
	return session.NewDBWrapper(&DB{gormDB: db})
}

type DB struct {
	gormDB *gorm.DB
}

func (db *DB) GetDB(ctx context.Context) *gorm.DB {
	return db.gormDB
}

func (db *DB) ConvertTx(ctx context.Context, tx *sql.Tx) *gorm.DB {
	gormTx := db.gormDB.Session(&gorm.Session{
		Context: ctx,
		NewDB:   true,
	})
	gormTx.Statement.ConnPool = tx
	return gormTx
}
