package transaction

import (
	"context"
	"database/sql"

	"gorm.io/gorm"

	"github.com/aeramu/sql-transaction/session"
)

type Session struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Session {
	return &Session{
		db: db,
	}
}

func (s *Session) getTx(ctx context.Context) (*gorm.DB, bool) {
	val := session.GetTx(ctx)
	tx, ok := val.(*sql.Tx)
	if !ok || tx == nil {
		return nil, false
	}
	session := s.db.Session(&gorm.Session{
		Context: ctx,
		NewDB:   true,
	})
	session.Statement.ConnPool = tx
	return session, true
}

func (s *Session) GetDB(ctx context.Context) *gorm.DB {
	tx, exist := s.getTx(ctx)
	if !exist {
		return s.db
	}
	return tx
}
