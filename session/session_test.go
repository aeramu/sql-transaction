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
	session Session
	db      DBWrapper[Executor]
	sqlDB   *sql.DB
}

// Setup test suite
func (s *SessionTestSuite) SetupTest() {
	db, err := sql.Open("sqlite3", ":memory:")
	s.Require().NoError(err)

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS models (id TEXT PRIMARY KEY)`)
	s.Require().NoError(err)

	s.sqlDB = db
	s.session = NewSession(db)
	s.db = NewDB(db)
}

func (s *SessionTestSuite) TearDownTest() {
	_, err := s.sqlDB.Exec(`DROP TABLE IF EXISTS models`)
	s.Require().NoError(err)
	s.sqlDB.Close()
}

func (s *SessionTestSuite) TestWithTransaction_transactionInjected() {
	err := s.session.WithTransaction(context.Background(), func(ctx context.Context) error {
		tx := GetTx(ctx)
		s.NotNil(tx)

		db := s.db.GetDB(ctx)
		s.NotNil(db)
		s.Equal(tx, db)

		return nil
	})

	s.NoError(err)
}

func (s *SessionTestSuite) TestWithTransaction_errReturned() {
	expectedErr := errors.New("test error")
	err := s.session.WithTransaction(context.Background(), func(ctx context.Context) error {
		return expectedErr
	})

	s.Error(err)
	s.ErrorIs(err, expectedErr)
}

func (s *SessionTestSuite) TestWithTransaction_transactionCommitted() {
	err := s.session.WithTransaction(context.Background(), func(ctx context.Context) error {
		db := s.db.GetDB(ctx)
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
	err := s.session.WithTransaction(context.Background(), func(ctx context.Context) error {
		db := s.db.GetDB(ctx)
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
	err := s.session.WithTransaction(context.Background(), func(ctx context.Context) error {
		outerTx := s.db.GetDB(ctx)
		err := s.session.WithTransaction(ctx, func(ctx context.Context) error {
			innerTx := s.db.GetDB(ctx)
			s.Equal(outerTx, innerTx)
			return nil
		})
		s.NoError(err)
		return nil
	})

	s.NoError(err)
}

func (s *SessionTestSuite) TestWithTransaction_doubleTransactionCommitted() {
	err := s.session.WithTransaction(context.Background(), func(ctx context.Context) error {
		outerDB := s.db.GetDB(ctx)
		_, err := outerDB.Exec("INSERT INTO models (id) VALUES (?)", "test-double-commit-1")
		s.NoError(err)

		err = s.session.WithTransaction(ctx, func(ctx context.Context) error {
			innerDB := s.db.GetDB(ctx)
			_, err := innerDB.Exec("INSERT INTO models (id) VALUES (?)", "test-double-commit-2")
			return err
		})
		s.NoError(err)
		return nil
	})

	s.NoError(err)

	var id1, id2 string
	err = s.sqlDB.QueryRow("SELECT id FROM models WHERE id = ?", "test-double-commit-1").Scan(&id1)
	s.NoError(err)
	s.Equal("test-double-commit-1", id1)

	err = s.sqlDB.QueryRow("SELECT id FROM models WHERE id = ?", "test-double-commit-2").Scan(&id2)
	s.NoError(err)
	s.Equal("test-double-commit-2", id2)
}

func (s *SessionTestSuite) TestWithTransaction_doubleTransactionRolledBack() {
	err := s.session.WithTransaction(context.Background(), func(ctx context.Context) error {
		outerDB := s.db.GetDB(ctx)
		_, err := outerDB.Exec("INSERT INTO models (id) VALUES (?)", "test-double-rollback-1")
		s.NoError(err)

		err = s.session.WithTransaction(ctx, func(ctx context.Context) error {
			innerDB := s.db.GetDB(ctx)
			_, err := innerDB.Exec("INSERT INTO models (id) VALUES (?)", "test-double-rollback-2")
			s.NoError(err)
			return errors.New("rollback error")
		})
		s.Error(err)
		return err
	})

	s.Error(err)

	var count int
	err = s.sqlDB.QueryRow("SELECT COUNT(*) FROM models WHERE id IN (?, ?)",
		"test-double-rollback-1", "test-double-rollback-2").Scan(&count)
	s.NoError(err)
	s.Equal(0, count)
}

func (s *SessionTestSuite) TestWithTransaction_panicRecovery() {
	s.Panics(func() {
		_ = s.session.WithTransaction(context.Background(), func(ctx context.Context) error {
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
