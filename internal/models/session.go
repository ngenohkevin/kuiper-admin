package models

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ngenohkevin/kuiper_admin/internal/database"
)

type Session struct {
	ID             string           `json:"id"`
	Token          string           `json:"token"`
	Data           json.RawMessage  `json:"data"`
	CreatedAt      pgtype.Timestamp `json:"created_at"`
	ExpiresAt      pgtype.Timestamp `json:"expires_at"`
	LastAccessedAt pgtype.Timestamp `json:"last_accessed_at"`
}

// GetAllSessions retrieves all sessions from the database
func GetAllSessions(db *database.DB) ([]Session, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
		SELECT id, token, data, created_at, expires_at, last_accessed_at
		FROM sessions
		ORDER BY created_at DESC
	`

	rows, err := db.Pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("error querying sessions: %w", err)
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var s Session
		if err := rows.Scan(&s.ID, &s.Token, &s.Data, &s.CreatedAt, &s.ExpiresAt, &s.LastAccessedAt); err != nil {
			return nil, fmt.Errorf("error scanning session row: %w", err)
		}
		sessions = append(sessions, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating session rows: %w", err)
	}

	return sessions, nil
}

// GetSessionByID retrieves a single session by ID
func GetSessionByID(db *database.DB, id string) (Session, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
		SELECT id, token, data, created_at, expires_at, last_accessed_at
		FROM sessions
		WHERE id = $1
	`

	var s Session
	err := db.Pool.QueryRow(ctx, query, id).Scan(
		&s.ID, &s.Token, &s.Data, &s.CreatedAt, &s.ExpiresAt, &s.LastAccessedAt,
	)
	if err != nil {
		return Session{}, fmt.Errorf("error finding session: %w", err)
	}

	return s, nil
}

// UpdateSession updates an existing session in the database
func UpdateSession(db *database.DB, id, token string, data json.RawMessage, expiresAt time.Time) (Session, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
		UPDATE sessions
		SET token = $2, data = $3, expires_at = $4, last_accessed_at = CURRENT_TIMESTAMP
		WHERE id = $1
		RETURNING id, token, data, created_at, expires_at, last_accessed_at
	`

	var s Session
	err := db.Pool.QueryRow(ctx, query, id, token, data, expiresAt).Scan(
		&s.ID, &s.Token, &s.Data, &s.CreatedAt, &s.ExpiresAt, &s.LastAccessedAt,
	)
	if err != nil {
		return Session{}, fmt.Errorf("error updating session: %w", err)
	}

	return s, nil
}

// DeleteSession deletes a session from the database
func DeleteSession(db *database.DB, id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Check if there are any reviews referencing this session
	var count int
	err := db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM reviews WHERE session_id = $1", id).Scan(&count)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("error checking reviews using this session: %w", err)
	}

	if count > 0 {
		return fmt.Errorf("cannot delete session: it is referenced by %d reviews", count)
	}

	// Delete the session
	_, err = db.Pool.Exec(ctx, "DELETE FROM sessions WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("error deleting session: %w", err)
	}

	return nil
}

// SearchSessions searches for sessions by token or ID
func SearchSessions(db *database.DB, searchQuery string) ([]Session, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
		SELECT id, token, data, created_at, expires_at, last_accessed_at
		FROM sessions
		WHERE id::text ILIKE $1 OR token ILIKE $1
		ORDER BY created_at DESC
	`

	rows, err := db.Pool.Query(ctx, query, "%"+searchQuery+"%")
	if err != nil {
		return nil, fmt.Errorf("error searching sessions: %w", err)
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var s Session
		if err := rows.Scan(&s.ID, &s.Token, &s.Data, &s.CreatedAt, &s.ExpiresAt, &s.LastAccessedAt); err != nil {
			return nil, fmt.Errorf("error scanning session row: %w", err)
		}
		sessions = append(sessions, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating session rows: %w", err)
	}

	return sessions, nil
}

// IsSessionExpired checks if a session has expired
func IsSessionExpired(session Session) bool {
	if !session.ExpiresAt.Valid {
		return false // Consider sessions without expiry as non-expired
	}
	return session.ExpiresAt.Time.Before(time.Now())
}

// GetDataAsMap returns the session data as a map
func (s Session) GetDataAsMap() (map[string]interface{}, error) {
	if len(s.Data) == 0 {
		return map[string]interface{}{}, nil
	}

	var data map[string]interface{}
	if err := json.Unmarshal(s.Data, &data); err != nil {
		return nil, err
	}
	return data, nil
}

// GetPrettyJSON returns the session data as a formatted JSON string
func (s Session) GetPrettyJSON() string {
	if len(s.Data) == 0 {
		return "{}"
	}

	var temp interface{}
	if err := json.Unmarshal(s.Data, &temp); err != nil {
		return string(s.Data)
	}

	pretty, err := json.MarshalIndent(temp, "", "  ")
	if err != nil {
		return string(s.Data)
	}

	return string(pretty)
}

// GetStatus returns a human-readable status of the session
func (s Session) GetStatus() string {
	if IsSessionExpired(s) {
		return "Expired"
	}
	return "Active"
}
