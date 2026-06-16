package store_test

import "testing"

func TestPrefsRoundTrip(t *testing.T) {
	s := newTestStore(t)
	if err := s.SetPref("timezone", "America/New_York"); err != nil {
		t.Fatal(err)
	}
	if err := s.SetPref("timezone", "Europe/London"); err != nil {
		t.Fatal(err)
	}
	v, err := s.GetPref("timezone")
	if err != nil || v != "Europe/London" {
		t.Fatalf("GetPref = %q err %v", v, err)
	}
	missing, err := s.GetPref("nope")
	if err != nil || missing != "" {
		t.Fatalf("missing pref should be empty string, got %q err %v", missing, err)
	}
}

func TestRemindersRoundTrip(t *testing.T) {
	s := newTestStore(t)
	id, err := s.AddReminder(10, 30)
	if err != nil || id <= 0 {
		t.Fatalf("AddReminder id=%d err=%v", id, err)
	}
	rs, err := s.Reminders()
	if err != nil || len(rs) != 1 || rs[0].MatchID != 10 || rs[0].LeadMinutes != 30 {
		t.Fatalf("Reminders = %+v err %v", rs, err)
	}
	if err := s.DeleteReminder(id); err != nil {
		t.Fatal(err)
	}
	rs, _ = s.Reminders()
	if len(rs) != 0 {
		t.Fatalf("after delete: %+v", rs)
	}
}
