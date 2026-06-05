package main

import (
	"database/sql"
	"errors"
	"path/filepath"

	_ "modernc.org/sqlite" // pure-Go SQLite driver, registers as "sqlite"
)

type Store struct {
	db *sql.DB
}

const schema = `
CREATE TABLE IF NOT EXISTS sessions (
	id              TEXT PRIMARY KEY,
	library_item_id TEXT,
	media_type      TEXT,
	display_title   TEXT,
	display_author  TEXT,
	duration        REAL,
	time_listening  REAL,
	start_time      REAL,
	current_pos     REAL,
	created_at      INTEGER,
	updated_at      INTEGER
);
CREATE INDEX IF NOT EXISTS idx_sessions_media ON sessions(media_type);

CREATE TABLE IF NOT EXISTS meta (
	key   TEXT PRIMARY KEY,
	value TEXT
);
`

func openStore(dir string) (*Store, error) {
	db, err := sql.Open("sqlite", filepath.Join(dir, "abs.db"))
	if err != nil {
		return nil, err
	}
	// SQLite handles one writer at a time; serialize to avoid "database is locked".
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000;`); err != nil {
		db.Close()
		return nil, err
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

// Ping verifies the database connection is alive (used by the healthcheck).
func (s *Store) Ping() error { return s.db.Ping() }

// upsertSessions inserts/updates a page of sessions in one transaction and
// returns how many rows were new or had a changed updated_at. The caller uses
// that count to detect when an incremental sync has reached already-synced data.
func (s *Store) upsertSessions(sessions []absSession) (changed int, err error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	check, err := tx.Prepare(`SELECT updated_at FROM sessions WHERE id=?`)
	if err != nil {
		return 0, err
	}
	defer check.Close()

	up, err := tx.Prepare(`
		INSERT INTO sessions
			(id, library_item_id, media_type, display_title, display_author,
			 duration, time_listening, start_time, current_pos, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET
			library_item_id=excluded.library_item_id,
			media_type=excluded.media_type,
			display_title=excluded.display_title,
			display_author=excluded.display_author,
			duration=excluded.duration,
			time_listening=excluded.time_listening,
			start_time=excluded.start_time,
			current_pos=excluded.current_pos,
			created_at=excluded.created_at,
			updated_at=excluded.updated_at`)
	if err != nil {
		return 0, err
	}
	defer up.Close()

	for _, ss := range sessions {
		if ss.ID == "" {
			continue
		}
		var prev int64
		switch e := check.QueryRow(ss.ID).Scan(&prev); {
		case errors.Is(e, sql.ErrNoRows):
			changed++
		case e != nil:
			return 0, e
		case prev != ss.UpdatedAt:
			changed++
		}
		title := ss.DisplayTitle
		if title == "" {
			title = ss.MediaMetadata.Title
		}
		if _, err = up.Exec(ss.ID, ss.LibraryItemID, ss.MediaType, title, ss.DisplayAuthor,
			ss.Duration, ss.TimeListening, ss.StartTime, ss.CurrentTime, ss.CreatedAt, ss.UpdatedAt); err != nil {
			return 0, err
		}
	}

	err = tx.Commit()
	return changed, err
}

// bookSessions returns every stored book session, for aggregation.
func (s *Store) bookSessions() ([]absSession, error) {
	rows, err := s.db.Query(`
		SELECT id, library_item_id, media_type, display_title, display_author,
		       duration, time_listening, start_time, current_pos, created_at, updated_at
		FROM sessions WHERE media_type='book'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []absSession
	for rows.Next() {
		var s absSession
		if err := rows.Scan(&s.ID, &s.LibraryItemID, &s.MediaType, &s.DisplayTitle, &s.DisplayAuthor,
			&s.Duration, &s.TimeListening, &s.StartTime, &s.CurrentTime, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// distinctBookIDs returns the set of library item ids that have book sessions.
func (s *Store) distinctBookIDs() ([]string, error) {
	rows, err := s.db.Query(`SELECT DISTINCT library_item_id FROM sessions WHERE media_type='book' AND library_item_id<>''`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (s *Store) sessionCount() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&n)
	return n, err
}

func (s *Store) getMeta(key string) (string, error) {
	var v string
	err := s.db.QueryRow(`SELECT value FROM meta WHERE key=?`, key).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return v, err
}

// getMetaBool reports whether the meta key is set to "1". Missing/unreadable
// keys read as false.
func (s *Store) getMetaBool(key string) bool {
	v, _ := s.getMeta(key)
	return v == "1"
}

func (s *Store) setMeta(key, value string) error {
	_, err := s.db.Exec(`INSERT INTO meta(key,value) VALUES(?,?)
		ON CONFLICT(key) DO UPDATE SET value=excluded.value`, key, value)
	return err
}
