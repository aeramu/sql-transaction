package transaction

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"gorm.io/gorm/logger"

	"github.com/aeramu/sql-transaction/session"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type TransactionTestSuite struct {
	suite.Suite
	gsession *Session
	gdb      *gorm.DB
	db       *sql.DB
	session  session.Session
}

type model struct {
	ID string
}

// Setup test suite
func (s *TransactionTestSuite) SetupTest() {
	db, err := sql.Open("sqlite3", ":memory:")
	s.Require().NoError(err)

	gdb, err := gorm.Open(sqlite.New(sqlite.Config{
		Conn: db,
	}))
	s.Require().NoError(err)
	gdb.Logger = logger.Default.LogMode(logger.Silent)

	err = gdb.AutoMigrate(&model{})
	s.Require().NoError(err)

	s.gdb = gdb
	s.gsession = New(gdb)
	s.db = db
	s.session = session.NewSession(db)
}

func (s *TransactionTestSuite) TestGetTx_noTx() {
	tx, exist := s.gsession.getTx(context.Background())
	s.False(exist)
	s.Nil(tx)
}

func (s *TransactionTestSuite) TestGetTx_txWrongType() {
	ctx := session.WithTx(context.Background(), "any")

	tx, exist := s.gsession.getTx(ctx)
	s.False(exist)
	s.Nil(tx)
}

func (s *TransactionTestSuite) TestGetTx_success() {
	tx, err := s.db.Begin()
	s.Require().NoError(err)
	ctx := session.WithTx(context.Background(), tx)

	db, exist := s.gsession.getTx(ctx)
	s.True(exist)
	s.NotNil(db)
	s.Equal(db.Statement.ConnPool, tx)
}

func (s *TransactionTestSuite) TestGetDB_returnDBWhenNoTx() {
	db := s.gsession.GetDB(context.Background())
	s.NotNil(db)
	s.Equal(db, s.gdb)
}

func (s *TransactionTestSuite) TestGetDB_returnTx() {
	tx, err := s.db.Begin()
	s.Require().NoError(err)
	ctx := session.WithTx(context.Background(), tx)

	db := s.gsession.GetDB(ctx)
	s.NotNil(db)
	s.Equal(db.Statement.ConnPool, tx)
}

func (s *TransactionTestSuite) TestWithTransaction_transactionInjected() {
	err := s.session.WithTransaction(context.Background(), func(ctx context.Context) error {
		tx := session.GetTx(ctx)
		s.Assert().NotNil(tx)

		db := s.gsession.GetDB(ctx)
		s.Assert().NotNil(db)
		s.Equal(tx, db.Statement.ConnPool)

		return nil
	})

	s.NoError(err)
}

func (s *TransactionTestSuite) TestWithTransaction_errPassed() {
	err := s.session.WithTransaction(context.Background(), func(ctx context.Context) error {
		return gorm.ErrRecordNotFound
	})

	s.Error(err)
	s.ErrorIs(err, gorm.ErrRecordNotFound)
}

func (s *TransactionTestSuite) TestWithTransaction_transacitonCommited() {
	data := model{ID: "test-transaciton-commited"}
	err := s.session.WithTransaction(context.Background(), func(ctx context.Context) error {
		res := s.gsession.GetDB(ctx).Create(&data)
		s.NoError(res.Error)
		return nil
	})

	s.NoError(err)

	var inserted model
	res := s.gdb.First(&inserted, "id = ?", data.ID)
	s.NoError(res.Error)
	s.Equal(data, inserted)
}

func (s *TransactionTestSuite) TestWithTransaction_transactionRolledBack() {
	data := model{ID: "test-transaciton-rollback"}
	err := s.session.WithTransaction(context.Background(), func(ctx context.Context) error {
		res := s.gsession.GetDB(ctx).Create(&data)
		s.NoError(res.Error)
		return errors.New("need to be rollback")
	})

	s.Error(err)

	var inserted model
	res := s.gdb.First(&inserted, "id = ?", data.ID)
	s.Error(res.Error)
	s.ErrorIs(res.Error, gorm.ErrRecordNotFound)
}

func (s *TransactionTestSuite) TestWithTransaction_doubleTransactionInjection() {
	err := s.session.WithTransaction(context.Background(), func(ctx context.Context) error {
		tx := s.gsession.GetDB(ctx)
		err := s.session.WithTransaction(ctx, func(ctx context.Context) error {
			childTx := s.gsession.GetDB(ctx)
			s.Assert().Equal(tx.Statement.ConnPool, childTx.Statement.ConnPool)
			return nil
		})
		s.NoError(err)
		return err
	})

	s.NoError(err)
}

func (s *TransactionTestSuite) TestWithTransaction_doubleTransactionCommitted() {
	data1 := model{ID: "test-double-transaction-commit-1"}
	data2 := model{ID: "test-double-transaction-commit-2"}
	err := s.session.WithTransaction(context.Background(), func(ctx context.Context) error {
		res := s.gsession.GetDB(ctx).Create(&data2)
		s.NoError(res.Error)
		err := s.session.WithTransaction(ctx, func(ctx context.Context) error {
			res := s.gsession.GetDB(ctx).Create(&data1)
			s.NoError(res.Error)
			return nil
		})
		s.NoError(err)
		return nil
	})

	s.NoError(err)

	var inserted1 model
	res := s.gdb.First(&inserted1, "id = ?", data1.ID)
	s.NoError(res.Error)
	s.Equal(data1, inserted1)

	var inserted2 model
	res = s.gdb.First(&inserted2, "id = ?", data2.ID)
	s.NoError(res.Error)
	s.Equal(data2, inserted2)
}

func (s *TransactionTestSuite) TestWithTransaction_doubleTransactionRolledBack() {
	data1 := model{ID: "test-double-transaction-rollback-1"}
	data2 := model{ID: "test-double-transaction-rollback-2"}
	err := s.session.WithTransaction(context.Background(), func(ctx context.Context) error {
		res := s.gsession.GetDB(ctx).Create(&data2)
		s.NoError(res.Error)
		err := s.session.WithTransaction(ctx, func(ctx context.Context) error {
			res := s.gsession.GetDB(ctx).Create(&data1)
			s.NoError(res.Error)
			return errors.New("need to be rollback")
		})
		s.Error(err)
		return err
	})

	s.Error(err)

	var inserted1 model
	res := s.gdb.First(&inserted1, "id = ?", data1.ID)
	s.Error(res.Error)
	s.ErrorIs(res.Error, gorm.ErrRecordNotFound)

	var inserted2 model
	res = s.gdb.First(&inserted2, "id = ?", data2.ID)
	s.Error(res.Error)
	s.ErrorIs(res.Error, gorm.ErrRecordNotFound)
}


func TestTransactionTestSuite(t *testing.T) {
	suite.Run(t, new(TransactionTestSuite))
}
