package state

import (
	"database/sql"
	"log"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

type Store struct {
	db *sql.DB
	mu sync.Mutex
}

func NewSQLite(path string) (*Store, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}

	Store := &Store{db: db}

	if err := db.Ping(); err != nil {
		return nil, err
	}
	log.Println("SQLite connection OK")
	if err := Store.initSchema(); err != nil {
		return nil, err
	}
	return Store, nil
}

func (s *Store) Exists(collection string) bool {
	var count int
	_ = s.db.QueryRow(
		`SELECT COUNT(1) FROM collection_state WHERE collection = ?`,
		collection,
	).Scan(&count)

	return count > 0
}

func (s *Store) Get(collection string) State {
	var st string
	err := s.db.QueryRow(
		`SELECT state FROM collection_state WHERE collection = ?`,
		collection,
	).Scan(&st)

	if err != nil {
		log.Println("Unable to get in sqllite", err)
		return "" // unknown
	}

	return State(st)
}

func (s *Store) Set(collection string, state State) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := s.db.Exec(`
		INSERT INTO collection_state(collection, state)
		VALUES (?, ?)
		ON CONFLICT(collection)
		DO UPDATE SET state=excluded.state, updated_at=CURRENT_TIMESTAMP
	`, collection, string(state)); err != nil {
		log.Println("Unable to set in sqlite", err)
	}

}

func (s *Store) initSchema() error {
	const schema = `
	CREATE TABLE IF NOT EXISTS collection_state (
		collection TEXT PRIMARY KEY,
		state TEXT NOT NULL,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`

	_, err := s.db.Exec(schema)
	return err
}
