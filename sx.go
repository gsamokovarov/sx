// Package sx provides helpers over the standard database/sql usage.
package sx

import (
	"database/sql"
	"fmt"
)

// Beginner begins transactions.
type Beginner interface {
	Begin() (*sql.Tx, error)
}

// Executor mimics the common behaviour between sql.DB and sql.Tx.
type Executor interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}

// Transactor can commit and rollback, on top of being able to execute queries.
// It can also begin nested transactions.
type Transactor interface {
	Beginner
	Executor

	Commit() error
	Rollback() error
}

// Transaction runs the function in an SQL transaction. Rollbacks will happen
// automatically if the function panics or it returns an error.
func Transaction(db Beginner, action func(Transactor) error) error {
	impl, err := db.Begin()
	if err != nil {
		return err
	}

	var tx Transactor
	switch db.(type) {
	case *nestableTransactor, *nestedTransactor:
		tx = &nestedTransactor{impl}
	default:
		tx = &nestableTransactor{impl}
	}

	// Rollback the transaction on panics in the action. Don't swallow the
	// panic, though, let it propagate.
	defer func(tx Transactor) {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}(tx)

	if err := action(tx); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// NewTransactor creates a new transactor.
func NewTransactor(v Executor) Transactor {
	switch tx := v.(type) {
	case *sql.DB:
		return &dbTransactor{tx}
	case *sql.Tx:
		return &nestableTransactor{tx}
	case *nestedTransactor:
		return &nestedTransactor{tx.Tx}
	default:
		panic(fmt.Sprintf("unsupported transactor type: %T", tx))
	}
}

// dbTransactor wraps a regular sql.DB pointer and promotes it to sx.Transactor.
type dbTransactor struct{ *sql.DB }

func (t *dbTransactor) Commit() error   { return nil }
func (t *dbTransactor) Rollback() error { return nil }

// nestableTransactor allows us to support helper database functions, that
// don't need to worry whether a transaction is already started or they should
// start a new one.
type nestableTransactor struct{ *sql.Tx }

func (t *nestableTransactor) Begin() (*sql.Tx, error) { return t.Tx, nil }

// nestedTransactor is used in nested transactions so the inner InTransaction
// block cannot commit the outer migration. It can revert it, though.
type nestedTransactor struct{ *sql.Tx }

func (t *nestedTransactor) Begin() (*sql.Tx, error) { return t.Tx, nil }
func (t *nestedTransactor) Commit() error           { return nil }
