package state

import (
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

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

func (s *Store) Touch(collection string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()

	_, err := s.db.Exec(`
		UPDATE collection_state
		SET last_accessed_at = ?
		WHERE collection = ?
	`, now, collection)

	if err != nil {
		log.Printf("Touch failed for %s: %v", collection, err)
	}
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

func (s *Store) ListHotOlderThan(d time.Duration) []string {
	seconds := int64(d.Seconds())
	rows, err := s.db.Query(`
    SELECT collection
    FROM collection_state
    WHERE state = 'HOT'
      AND last_accessed_at IS NOT NULL
      AND last_accessed_at < DATETIME('now', ?)
`, fmt.Sprintf("-%d seconds", seconds))
	if err != nil {
		return nil
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var c string
		rows.Scan(&c)
		out = append(out, c)
	}
	// fmt.Println(`SELECT collection
	// 	FROM collection_state
	// 	WHERE state = 'HOT'
	// 	AND last_accessed_at IS NOT NULL
	// 	AND last_accessed_at < DATETIME('now', ?)
	// `, "-"+d.String())
	fmt.Println(out)
	return out
}

func (s *Store) WasRecentlyAccessed(collection string, d time.Duration) bool {
	var count int
	cutoff := time.Now().UTC().Add(-d)

	err := s.db.QueryRow(`
		SELECT COUNT(1)
		FROM collection_state
		WHERE collection = ?
		  AND last_accessed_at IS NOT NULL
		  AND last_accessed_at > ?
	`, collection, cutoff).Scan(&count)

	if err != nil {
		return false
	}

	return count > 0
}

func (s *Store) initSchema() error {
	const schema = `
	CREATE TABLE IF NOT EXISTS collection_state (
		collection TEXT PRIMARY KEY,
		state TEXT NOT NULL,
		last_accessed_at DATETIME,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`

	_, err := s.db.Exec(schema)
	return err
}

func (s *Store) CountByState() map[State]int {
	rows, err := s.db.Query(`
		SELECT state, COUNT(state)
		FROM collection_state
		GROUP BY state
	`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	out := make(map[State]int)
	for rows.Next() {
		var st string
		var count int
		rows.Scan(&st, &count)
		out[State(st)] = count
	}
	return out
}
