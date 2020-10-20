package users

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"tickets/generated"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Store struct{ db *sql.DB }

func NewInmemSQLStore() (*Store, error) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, fmt.Errorf(
			"opening in-memory SQLite3 database: %w", err,
		)
	}

	tx, err := db.BeginTx(context.Background(), &sql.TxOptions{
		Isolation: sql.LevelSerializable,
		ReadOnly:  false,
	})
	if err != nil {
		return nil, fmt.Errorf("initializing transaction: %w", err)
	}

	if _, err := tx.Exec(
		`CREATE TABLE metadata (k INTEGER PRIMARY KEY, v TEXT NOT NULL);`,
	); err != nil {
		return nil, fmt.Errorf("creating users table")
	}
	if _, err := tx.Exec(
		`INSERT INTO metadata (k,v) VALUES (0,'0')`,
	); err != nil {
		return nil, fmt.Errorf("setting projection version metadata value")
	}
	if _, err := tx.Exec(
		`CREATE TABLE users(id TEXT PRIMARY KEY, name TEXT NOT NULL);`,
	); err != nil {
		return nil, fmt.Errorf("creating users table")
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("committing transaction: %w", err)
	}

	return &Store{
		db: db,
	}, nil
}

type transaction struct{ tx *sql.Tx }

func (t transaction) Commit() {
	if err := t.tx.Commit(); err != nil {
		log.Printf("committing transaction %p: %s", t.tx, err)
	}
}

func (t transaction) Rollback() {
	if err := t.tx.Rollback(); err != nil {
		log.Printf("rolling back transaction %p: %s", t.tx, err)
	}
}

type transactionRead struct{ tx *sql.Tx }

func (t transactionRead) Complete() {
	if err := t.tx.Commit(); err != nil {
		log.Printf(
			"committing read-only transaction %p: %s",
			t.tx, err,
		)
	}
}

func (s *Store) NewTransactionReadWriter() generated.StoreTransactionReadWriter {
	t, err := s.db.BeginTx(context.Background(), &sql.TxOptions{
		Isolation: sql.LevelSerializable,
		ReadOnly:  false,
	})
	if err != nil {
		panic(err)
	}
	return transaction{t}
}

func (s *Store) NewTransactionReader() generated.StoreTransactionReader {
	t, err := s.db.BeginTx(context.Background(), &sql.TxOptions{
		ReadOnly: true,
	})
	if err != nil {
		panic(err)
	}
	return transactionRead{t}
}

func (s *Store) UpdateProjectionVersion(
	ctx context.Context,
	tx generated.TransactionWriter,
	v generated.EventlogVersion,
) (err error) {
	_, err = tx.(transaction).tx.Exec(
		`UPDATE metadata SET v=? WHERE k=0`, v,
	)
	return
}

// ProjectionVersion returns the current projection version.
// Returns an empty string if the projection wasn't initialized yet.
// In case an empty string is returned the service will fallback
// to the begin offset version of the eventlog.
func (s *Store) ProjectionVersion(
	ctx context.Context,
	tx generated.TransactionReader,
) (v generated.EventlogVersion, err error) {
	err = tx.(transaction).tx.
		QueryRow("SELECT v FROM metadata WHERE k=0").
		Scan(&v)
	return
}

// ApplyEventUserCreated applies event UserCreated to the projection.
// The projection must update its local projection version
// to the one that is provided.
func (s *Store) ApplyEventUserCreated(
	ctx context.Context,
	tx generated.TransactionWriter,
	v generated.EventlogVersion,
	tm time.Time,
	e generated.EventUserCreated,
) error {
	if _, err := tx.(transaction).tx.Exec(
		`INSERT INTO users (id,name) VALUES (?,?)`,
		e.Id, e.Name,
	); err != nil {
		return fmt.Errorf("updating projection version: %w", err)
	}
	return nil
}
