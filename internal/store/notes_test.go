package store_test

import (
	"testing"

	"touchline/internal/model"
)

func TestNotesCRUD(t *testing.T) {
	s := newTestStore(t)
	id, err := s.AddNote(model.Note{SubjectType: "team", SubjectID: 21, Body: "Watch the press"})
	if err != nil || id <= 0 {
		t.Fatalf("AddNote id=%d err=%v", id, err)
	}
	notes, err := s.NotesBySubject("team", 21)
	if err != nil || len(notes) != 1 || notes[0].Body != "Watch the press" {
		t.Fatalf("NotesBySubject = %+v err %v", notes, err)
	}
	if notes[0].CreatedAt.IsZero() {
		t.Fatal("CreatedAt should be set")
	}
	if err := s.DeleteNote(id); err != nil {
		t.Fatal(err)
	}
	notes, _ = s.NotesBySubject("team", 21)
	if len(notes) != 0 {
		t.Fatalf("after delete: %+v", notes)
	}
}
