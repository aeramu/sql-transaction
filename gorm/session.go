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

func getTx(ctx context.Context) (*gorm.DB, bool) {
	tx, ok := ctx.Value(txKey{}).(*gorm.DB)
	if !ok || tx == nil {
		return nil, false
	}
	return tx, true
}

func (s *DB) WithTransaction(ctx context.Context, f func (ctx context.Context) error) error {
	tx, txExist := getTx(ctx)
	if !txExist {
		tx = s.db.WithContext(ctx).Begin(s.txOpt)
		if tx.Error != nil {
			return tx.Error
		}
		ctx = context.WithValue(ctx, txKey{}, tx)
	}
	err := f(ctx)
	if err != nil {
		if !txExist {
			tx.Rollback()
		}
		return err
	}
	if !txExist {
		return tx.Commit().Error
	}
	return nil
}

func (s *DB) GetDB(ctx context.Context) *gorm.DB {
	tx, exist := getTx(ctx)
	if !exist {
		return s.db
	}
	return tx
}
