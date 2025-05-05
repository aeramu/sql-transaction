package session

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/suite"
)

type WrapperTestSuite struct {
	suite.Suite
	db      *sql.DB
	wrapper DBWrapper[Executor]
}

func (s *WrapperTestSuite) SetupTest() {
	db, err := sql.Open("sqlite3", ":memory:")
	s.Require().NoError(err)
	s.db = db
	s.wrapper = NewDB(db)
}

func (s *WrapperTestSuite) TearDownTest() {
	s.db.Close()
}

func (s *WrapperTestSuite) TestGetDB_NoTransaction() {
	// When no transaction in context, should return regular db
	result := s.wrapper.GetDB(context.Background())
	s.Equal(s.db, result)
}

func (s *WrapperTestSuite) TestGetDB_WrongTypeTransaction() {
	// Create context with wrong type value (string instead of *sql.Tx)
	ctx := WithTx(context.Background(), "tx")

	// Should return regular db since transaction type is invalid
	result := s.wrapper.GetDB(ctx)
	s.Equal(s.db, result)
}

func (s *WrapperTestSuite) TestGetDB_WithTransaction() {
	// Start a transaction
	tx, err := s.db.Begin()
	s.Require().NoError(err)
	defer tx.Rollback()

	// Create context with transaction
	ctx := WithTx(context.Background(), tx)

	// Should return the transaction
	result := s.wrapper.GetDB(ctx)
	s.Equal(tx, result)
}

func TestWrapperTestSuite(t *testing.T) {
	suite.Run(t, new(WrapperTestSuite))
}
