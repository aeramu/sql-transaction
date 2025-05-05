package session

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/suite"
)

type SessionTestSuite struct {
	suite.Suite
	db    Session
	sqlDB *sql.DB
}

// Setup test suite
func (s *SessionTestSuite) SetupTest() {
	db, err := sql.Open("sqlite3", ":memory:")
	s.Require().NoError(err)

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS models (id TEXT PRIMARY KEY)`)
	s.Require().NoError(err)

	s.sqlDB = db
	s.db = NewSession(db)
}

func (s *SessionTestSuite) TearDownTest() {
	_, err := s.sqlDB.Exec(`DROP TABLE IF EXISTS models`)
	s.Require().NoError(err)
	s.sqlDB.Close()
}

func (s *SessionTestSuite) TestGetTx_noTx() {
	tx, exist := getTx(context.Background())
	s.False(exist)
	s.Nil(tx)
}

func (s *SessionTestSuite) TestGetTx_txWrongType() {
	ctx := WithTx(context.Background(), nil)
	ctx = context.WithValue(ctx, txKey{}, "any")
	tx, exist := getTx(ctx)
	s.False(exist)
	s.Nil(tx)
}

func (s *SessionTestSuite) TestGetTx_success() {
	tx, err := s.sqlDB.Begin()
	s.Require().NoError(err)
	defer tx.Rollback()

	ctx := WithTx(context.Background(), tx)
	gotTx, exist := getTx(ctx)
	s.True(exist)
	s.NotNil(gotTx)
	s.Equal(tx, gotTx)
}

func (s *SessionTestSuite) TestGetDB_returnDBWhenNoTx() {
	db := s.db.(*session).GetDB(context.Background())
	s.NotNil(db)
	s.Equal(s.sqlDB, db)
}

func (s *SessionTestSuite) TestGetDB_returnTx() {
	tx, err := s.sqlDB.Begin()
	s.Require().NoError(err)
	defer tx.Rollback()

	ctx := WithTx(context.Background(), tx)
	db := s.db.(*session).GetDB(ctx)
	s.NotNil(db)
	s.Equal(tx, db)
}

func (s *SessionTestSuite) TestWithTransaction_transactionInjected() {
	err := s.db.WithTransaction(context.Background(), func(ctx context.Context) error {
		tx := GetTx(ctx)
		s.NotNil(tx)

		db := s.db.(*session).GetDB(ctx)
		s.NotNil(db)
		s.Equal(tx, db)

		return nil
	})

	s.NoError(err)
}

func (s *SessionTestSuite) TestWithTransaction_errPassed() {
	expectedErr := errors.New("test error")
	err := s.db.WithTransaction(context.Background(), func(ctx context.Context) error {
		return expectedErr
	})

	s.Error(err)
	s.ErrorIs(err, expectedErr)
}

func (s *SessionTestSuite) TestWithTransaction_transactionCommitted() {
	err := s.db.WithTransaction(context.Background(), func(ctx context.Context) error {
		db := s.db.(*session).GetDB(ctx)
		_, err := db.Exec("INSERT INTO models (id) VALUES (?)", "test-commit")
		return err
	})

	s.NoError(err)

	var id string
	err = s.sqlDB.QueryRow("SELECT id FROM models WHERE id = ?", "test-commit").Scan(&id)
	s.NoError(err)
	s.Equal("test-commit", id)
}

func (s *SessionTestSuite) TestWithTransaction_transactionRolledBack() {
	err := s.db.WithTransaction(context.Background(), func(ctx context.Context) error {
		db := s.db.(*session).GetDB(ctx)
		_, err := db.Exec("INSERT INTO models (id) VALUES (?)", "test-rollback")
		s.NoError(err)
		return errors.New("rollback error")
	})

	s.Error(err)

	var count int
	err = s.sqlDB.QueryRow("SELECT COUNT(*) FROM models WHERE id = ?", "test-rollback").Scan(&count)
	s.NoError(err)
	s.Equal(0, count)
}

func (s *SessionTestSuite) TestWithTransaction_doubleTransactionInjection() {
	err := s.db.WithTransaction(context.Background(), func(ctx context.Context) error {
		outerTx := s.db.(*session).GetDB(ctx)
		err := s.db.WithTransaction(ctx, func(ctx context.Context) error {
			innerTx := s.db.(*session).GetDB(ctx)
			s.Equal(outerTx, innerTx)
			return nil
		})
		s.NoError(err)
		return nil
	})

	s.NoError(err)
}

func (s *SessionTestSuite) TestWithTransaction_panicRecovery() {
	s.Panics(func() {
		_ = s.db.WithTransaction(context.Background(), func(ctx context.Context) error {
			panic("test panic")
		})
	})

	// Verify the transaction was rolled back
	var count int
	err := s.sqlDB.QueryRow("SELECT COUNT(*) FROM models").Scan(&count)
	s.NoError(err)
	s.Equal(0, count)
}

func TestSessionTestSuite(t *testing.T) {
	suite.Run(t, new(SessionTestSuite))
}
