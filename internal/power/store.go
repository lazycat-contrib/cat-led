package power

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	_ "github.com/lib-x/entsqlite"
	"net/url"
	"path/filepath"
)

type Store interface {
	Load(context.Context) (State, error)
	Save(context.Context, State) error
}

type SQLStore struct{ db *sql.DB }

// OpenStore adds one singleton table to the application's existing SQLite file.
// FULL synchronous commits make the pre-shutdown state durable before poweroff.
func OpenStore(path string) (*SQLStore, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	u := url.URL{Scheme: "file", Path: absolute}
	db, err := sql.Open("sqlite3", u.String()+"?_pragma=journal_mode(WAL)&_pragma=synchronous(FULL)&_pragma=busy_timeout(10000)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err = db.Exec(`CREATE TABLE IF NOT EXISTS rtc_power_plan (id INTEGER PRIMARY KEY CHECK(id=1), payload TEXT NOT NULL)`); err != nil {
		db.Close()
		return nil, err
	}
	return &SQLStore{db}, nil
}
func (s *SQLStore) Close() error { return s.db.Close() }
func (s *SQLStore) Load(ctx context.Context) (State, error) {
	var data string
	state := State{Phase: "disabled"}
	err := s.db.QueryRowContext(ctx, "SELECT payload FROM rtc_power_plan WHERE id=1").Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return state, nil
	}
	if err != nil {
		return state, err
	}
	err = json.Unmarshal([]byte(data), &state)
	return state, err
}
func (s *SQLStore) Save(ctx context.Context, state State) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, "INSERT INTO rtc_power_plan(id,payload) VALUES(1,?) ON CONFLICT(id) DO UPDATE SET payload=excluded.payload", string(data))
	return err
}
