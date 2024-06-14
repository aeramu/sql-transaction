package transaction

import (
	"context"
    "errors"
    "gorm.io/gorm/logger"
    "testing"

	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type TransactionTestSuite struct {
	suite.Suite
	db *DB
	gdb *gorm.DB
}

type model struct {
	ID string
}

// Setup test suite
func (s *TransactionTestSuite) SetupTest() {
	gdb, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.New(nil, logger.Config{
            LogLevel:                  logger.Silent,
        }),
	})
	s.Require().NoError(err)
	
	err = gdb.AutoMigrate(&model{})
	s.Require().NoError(err)

	s.gdb = gdb
	s.db = New(gdb)
}


func (s *TransactionTestSuite) TestGetTx_noTx() {
	tx, exist := getTx(context.Background())
	s.False(exist)
	s.Nil(tx)
}

func (s *TransactionTestSuite) TestGetTx_txWrongType() {
	ctx := context.WithValue(context.Background(), txKey{}, "any")
	tx, exist := getTx(ctx)
	s.False(exist)
	s.Nil(tx)
}

func (s *TransactionTestSuite) TestGetTx_success() {
	tx := s.gdb.Begin()
	ctx := context.WithValue(context.Background(), txKey{}, tx)
	db, exist := getTx(ctx)
	s.True(exist)
	s.NotNil(db)
	s.Equal(db, tx)
}

func (s *TransactionTestSuite) TestGetDB_returnDBWhenNoTx() {
	db := s.db.GetDB(context.Background())
	s.NotNil(db)
	s.Equal(db, s.gdb)
}

func (s *TransactionTestSuite) TestGetDB_returnTx() {
	tx := s.gdb.Begin()
	ctx := context.WithValue(context.Background(), txKey{}, tx)
	db := s.db.GetDB(ctx)
	s.NotNil(db)
	s.Equal(db, tx)
}

func (s *TransactionTestSuite) TestWithTransaction_transactionInjected() {
	err := s.db.WithTransaction(context.Background(), func(ctx context.Context) error {
		tx := ctx.Value(txKey{})
		s.Assert().NotNil(tx)
		
		db := s.db.GetDB(ctx)
		s.Assert().NotNil(db)
		s.Equal(tx, db)
		
		return nil
	})

	s.NoError(err)
}

func (s *TransactionTestSuite) TestWithTransaction_errPassed() {
	err := s.db.WithTransaction(context.Background(), func(ctx context.Context) error {
		return gorm.ErrRecordNotFound
	})

	s.Error(err)
	s.ErrorIs(err, gorm.ErrRecordNotFound)
}

func (s *TransactionTestSuite) TestWithTransaction_transacitonCommited() {
	data := model{ID: "test-transaciton-commited"}
	err := s.db.WithTransaction(context.Background(), func(ctx context.Context) error {
		res := s.db.GetDB(ctx).Create(&data)
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
	err := s.db.WithTransaction(context.Background(), func(ctx context.Context) error {
		res := s.db.GetDB(ctx).Create(&data)
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
	err := s.db.WithTransaction(context.Background(), func(ctx context.Context) error {
		tx := s.db.GetDB(ctx)
		err := s.db.WithTransaction(ctx, func(ctx context.Context) error {
			childTx := s.db.GetDB(ctx)
			s.Assert().Equal(tx, childTx)
			return nil
		})
		s.NoError(err)
		return err
	})

	s.NoError(err)
}

func TestTransactionTestSuite(t *testing.T) {
	suite.Run(t, new(TransactionTestSuite))
}