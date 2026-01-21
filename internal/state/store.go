package state

// import (
// 	"database/sql"

type State string

const (
	Hot     State = "HOT"
	Cold    State = "COLD"
	Loading State = "LOADING"
)

// 	_ "github.com/mattn/go-sqlite3"
// )

// type Store struct {
// 	DB *sql.DB
// }

// func New(path string) (*Store, error) {
// 	db, err := sql.Open("sqlite3", path)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &Store{DB: db}, nil
// }
