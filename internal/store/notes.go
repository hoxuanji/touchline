package store

import (
	"time"

	"touchline/internal/model"
)

func (s *Store) AddNote(n model.Note) (int, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.Exec(
		`INSERT INTO notes (subject_type,subject_id,body,created_at,updated_at) VALUES (?,?,?,?,?)`,
		n.SubjectType, n.SubjectID, n.Body, now, now)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return int(id), err
}

func (s *Store) NotesBySubject(subjectType string, subjectID int) ([]model.Note, error) {
	rows, err := s.db.Query(
		`SELECT id,subject_type,subject_id,body,created_at,updated_at FROM notes WHERE subject_type = ? AND subject_id = ? ORDER BY id DESC`,
		subjectType, subjectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Note
	for rows.Next() {
		var n model.Note
		var created, updated string
		if err := rows.Scan(&n.ID, &n.SubjectType, &n.SubjectID, &n.Body, &created, &updated); err != nil {
			return nil, err
		}
		n.CreatedAt, _ = time.Parse(time.RFC3339, created)
		n.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
		out = append(out, n)
	}
	return out, rows.Err()
}

func (s *Store) DeleteNote(id int) error {
	_, err := s.db.Exec(`DELETE FROM notes WHERE id = ?`, id)
	return err
}
