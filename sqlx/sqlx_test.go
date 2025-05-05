package transaction

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/aeramu/sql-transaction/session"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/suite"
)

type TransactionTestSuite struct {
	suite.Suite
	wrapper session.DBWrapper[Executor]
	sqlxDB   *sqlx.DB
	db       *sql.DB
	session  session.Session
}

type model struct {
	ID string `db:"id"`
}

// Setup test suite
func (s *TransactionTestSuite) SetupTest() {
	db, err := sql.Open("sqlite3", ":memory:")
	s.Require().NoError(err)

	sqlxDB := sqlx.NewDb(db, "sqlite3")
	s.Require().NoError(err)

	_, err = sqlxDB.Exec(`CREATE TABLE model (id TEXT PRIMARY KEY)`)
	s.Require().NoError(err)

	s.sqlxDB = sqlxDB
	s.wrapper = New(sqlxDB)
	s.db = db
	s.session = session.NewSession(db)
}

func (s *TransactionTestSuite) TestGetDB_returnDBWhenNoTx() {
	db := s.wrapper.GetDB(context.Background())
	s.NotNil(db)
	s.Equal(s.sqlxDB, db)
}

func (s *TransactionTestSuite) TestGetDB_returnTx() {
	tx, err := s.db.Begin()
	s.Require().NoError(err)
	ctx := session.WithTx(context.Background(), tx)

	db := s.wrapper.GetDB(ctx)
	s.NotNil(db)
	s.Equal(tx, db.(*sqlx.Tx).Tx)
}

func (s *TransactionTestSuite) TestWithTransaction_transactionInjected() {
	err := s.session.WithTransaction(context.Background(), func(ctx context.Context) error {
		tx := session.GetTx(ctx)
		s.Assert().NotNil(tx)

		db := s.wrapper.GetDB(ctx)
		s.Assert().NotNil(db)
		s.Equal(tx, db.(*sqlx.Tx).Tx)

		return nil
	})

	s.NoError(err)
}

func (s *TransactionTestSuite) TestWithTransaction_errPassed() {
	expectedErr := errors.New("some error")
	err := s.session.WithTransaction(context.Background(), func(ctx context.Context) error {
		return expectedErr
	})

	s.Error(err)
	s.ErrorIs(err, expectedErr)
}

func (s *TransactionTestSuite) TestWithTransaction_transacitonCommited() {
	data := model{ID: "test-transaciton-commited"}
	err := s.session.WithTransaction(context.Background(), func(ctx context.Context) error {
		_, err := s.wrapper.GetDB(ctx).Exec(`INSERT INTO model (id) VALUES (?)`, data.ID)
		s.NoError(err)
		return nil
	})

	s.NoError(err)

	var inserted model
	err = s.sqlxDB.Get(&inserted, `SELECT * FROM model WHERE id = ?`, data.ID)
	s.NoError(err)
	s.Equal(data, inserted)
}

func (s *TransactionTestSuite) TestWithTransaction_transactionRolledBack() {
	data := model{ID: "test-transaciton-rollback"}
	err := s.session.WithTransaction(context.Background(), func(ctx context.Context) error {
		_, err := s.wrapper.GetDB(ctx).Exec(`INSERT INTO model (id) VALUES (?)`, data.ID)
		s.NoError(err)
		return errors.New("need to be rollback")
	})

	s.Error(err)

	var inserted model
	err = s.sqlxDB.Get(&inserted, `SELECT * FROM model WHERE id = ?`, data.ID)
	s.Error(err)
	s.ErrorIs(err, sql.ErrNoRows)
}

func (s *TransactionTestSuite) TestWithTransaction_doubleTransactionInjection() {
	err := s.session.WithTransaction(context.Background(), func(ctx context.Context) error {
		tx := s.wrapper.GetDB(ctx)
		err := s.session.WithTransaction(ctx, func(ctx context.Context) error {
			childTx := s.wrapper.GetDB(ctx)
			s.Assert().Equal(tx.(*sqlx.Tx).Tx, childTx.(*sqlx.Tx).Tx)
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
		_, err := s.wrapper.GetDB(ctx).Exec(`INSERT INTO model (id) VALUES (?)`, data2.ID)
		s.NoError(err)
		err = s.session.WithTransaction(ctx, func(ctx context.Context) error {
			_, err := s.wrapper.GetDB(ctx).Exec(`INSERT INTO model (id) VALUES (?)`, data1.ID)
			s.NoError(err)
			return nil
		})
		s.NoError(err)
		return nil
	})

	s.NoError(err)

	var inserted1 model
	err = s.sqlxDB.Get(&inserted1, `SELECT * FROM model WHERE id = ?`, data1.ID)
	s.NoError(err)
	s.Equal(data1, inserted1)

	var inserted2 model
	err = s.sqlxDB.Get(&inserted2, `SELECT * FROM model WHERE id = ?`, data2.ID)
	s.NoError(err)
	s.Equal(data2, inserted2)
}

func (s *TransactionTestSuite) TestWithTransaction_doubleTransactionRolledBack() {
	data1 := model{ID: "test-double-transaction-rollback-1"}
	data2 := model{ID: "test-double-transaction-rollback-2"}
	err := s.session.WithTransaction(context.Background(), func(ctx context.Context) error {
		_, err := s.wrapper.GetDB(ctx).Exec(`INSERT INTO model (id) VALUES (?)`, data2.ID)
		s.NoError(err)
		err = s.session.WithTransaction(ctx, func(ctx context.Context) error {
			_, err := s.wrapper.GetDB(ctx).Exec(`INSERT INTO model (id) VALUES (?)`, data1.ID)
			s.NoError(err)
			return errors.New("need to be rollback")
		})
		s.Error(err)
		return err
	})

	s.Error(err)

	var inserted1 model
	err = s.sqlxDB.Get(&inserted1, `SELECT * FROM model WHERE id = ?`, data1.ID)
	s.Error(err)
	s.ErrorIs(err, sql.ErrNoRows)

	var inserted2 model
	err = s.sqlxDB.Get(&inserted2, `SELECT * FROM model WHERE id = ?`, data2.ID)
	s.Error(err)
	s.ErrorIs(err, sql.ErrNoRows)
}

func TestTransactionTestSuite(t *testing.T) {
	suite.Run(t, new(TransactionTestSuite))
}
