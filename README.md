# SX /sĕks/

> Contemplate or dwell on one's own success or another's misfortune with
> smugness or malignant pleasure.

SX is a Golang library that introduces nested `database/sql` transaction
support.

## [Issue 7898](https://github.com/golang/go/issues/7898)

Working with nested transactions in Golang's `database/sql` package is
problematic, because you have to distinquish between non-transaction and
transaction DB connections as they have different interfaces.

There is a long-standing issue in [Golang's issue tracker](https://github.com/golang/go/issues/7898),
but it's been 4 years and we still don't have the new API that supports nested
transactions.

SX tries to solve this issue by offering a new API that wraps both
non-transaction and transaction connections under one umbreala and offering a
single function, `sx.Transaction` to ensure a transaction, no matter if it was
open before.

```go
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
```

Currently, the nested transactions don't do save-states and a revert in a
nested transaction will rever the outer one, but this is easily solveable,
should you need to.

## Usage

```go
db, err := sql.Open(...)
if err != nil {
	// Handle error.
}

// Wrap the connection in `sx.Transactor`. `sx.NewTransactor` can work with
// both `*sql.DB`  and `*sql.Tx`.
tx := sx.NewTransactor(db)

err := sx.Transaction(tx, func(tx sx.Transactor) error {
	// Return `error` here to reverse the transaction. Any nested
	// `sx.Transaction` calls will reuse the same transaction.


	// Return `nil` on the outermost transaction to commit it. Returning
	// `nil` in nested transactions wont do a thing.
	return nil
})
```
