package tags

import (
	"database/sql"
	"fmt"

	"time-tracker/internal/shared/database"
)

type TagRepository struct {
	db *database.DB
}

func NewTagRepository(db *database.DB) *TagRepository {
	return &TagRepository{db: db}
}

func (r *TagRepository) Create(input *TagCreate) (*Tag, error) {
	res, err := r.db.Exec(
		`INSERT INTO tags (name, color, created_at) VALUES (?, ?, strftime('%Y-%m-%dT%H:%M:%SZ','now'))`,
		input.Name, input.Color,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert tag: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}
	return r.GetByID(id)
}

func (r *TagRepository) GetByID(id int64) (*Tag, error) {
	var t Tag
	err := r.db.QueryRow(`SELECT id, name, color, created_at FROM tags WHERE id = ?`, id).
		Scan(&t.ID, &t.Name, &t.Color, &t.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query tag: %w", err)
	}
	return &t, nil
}

func (r *TagRepository) List() ([]Tag, error) {
	rows, err := r.db.Query(`SELECT id, name, color, created_at FROM tags ORDER BY name ASC`)
	if err != nil {
		return nil, fmt.Errorf("failed to query tags: %w", err)
	}
	defer rows.Close()

	out := []Tag{}
	for rows.Next() {
		var t Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.Color, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan tag: %w", err)
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("tags rows error: %w", err)
	}

	return out, nil
}

func (r *TagRepository) AssignToSession(sessionID int64, tagIDs []int64) error {
	for _, tagID := range tagIDs {
		_, err := r.db.Exec(
			`INSERT OR IGNORE INTO session_tags (session_id, tag_id) VALUES (?, ?)`,
			sessionID, tagID,
		)
		if err != nil {
			return fmt.Errorf("failed to assign tag %d to session %d: %w", tagID, sessionID, err)
		}
	}
	return nil
}

func (r *TagRepository) RemoveFromSession(sessionID, tagID int64) error {
	res, err := r.db.Exec(
		`DELETE FROM session_tags WHERE session_id = ? AND tag_id = ?`,
		sessionID, tagID,
	)
	if err != nil {
		return fmt.Errorf("failed to remove tag %d from session %d: %w", tagID, sessionID, err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check remove result: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("session-tag association not found")
	}

	return nil
}

func (r *TagRepository) ListForSession(sessionID int64) ([]Tag, error) {
	rows, err := r.db.Query(
		`SELECT t.id, t.name, t.color, t.created_at
			FROM tags t
			INNER JOIN session_tags st ON st.tag_id = t.id
			WHERE st.session_id = ?
			ORDER BY t.name ASC`,
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query session tags: %w", err)
	}
	defer rows.Close()

	out := []Tag{}
	for rows.Next() {
		var t Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.Color, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan session tag: %w", err)
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("tags rows error: %w", err)
	}
	return out, nil
}
