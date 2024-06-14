package transaction

import (
	"context"
	"database/sql"
	"gorm.io/gorm"
)

type DB struct {
	db *gorm.DB
	txOpt *sql.TxOptions
}

func New(db *gorm.DB) *DB {
	return &DB{
		db: db,
	}
}

type txKey struct {}

func getTx(ctx context.Context) *gorm.DB {
	return ctx.Value(txKey{}).(*gorm.DB)
}

func (s *DB) WithTransaction(ctx context.Context, f func (ctx context.Context) error) error {
	tx := getTx(ctx)
	if tx == nil {
		tx = s.db.WithContext(ctx).Begin(s.txOpt)
		if tx.Error != nil {
			return tx.Error
		}
		ctx = context.WithValue(ctx, txKey{}, tx)
	}
	err := f(ctx)
	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func (s *DB) GetDB(ctx context.Context) *gorm.DB {
	tx := getTx(ctx)
	if tx == nil {
		return s.db
	}
	return tx
}
