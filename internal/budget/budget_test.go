package budget_test

import (
	"testing"
	"time"

	"touchline/internal/budget"
)

func TestBudgetAllowsUpToLimitPerDay(t *testing.T) {
	day := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)
	b := budget.New(3, func() time.Time { return day })
	for i := 0; i < 3; i++ {
		if !b.Allow() {
			t.Fatalf("call %d should be allowed", i)
		}
	}
	if b.Allow() {
		t.Fatal("4th call should be denied")
	}
	if b.Remaining() != 0 {
		t.Fatalf("Remaining = %d, want 0", b.Remaining())
	}
}

func TestBudgetResetsNextDay(t *testing.T) {
	now := time.Date(2026, 6, 16, 23, 0, 0, 0, time.UTC)
	b := budget.New(1, func() time.Time { return now })
	if !b.Allow() {
		t.Fatal("first call allowed")
	}
	if b.Allow() {
		t.Fatal("second call same day denied")
	}
	now = now.AddDate(0, 0, 1)
	if !b.Allow() {
		t.Fatal("call on new day should be allowed")
	}
}

func TestZeroLimitDeniesAll(t *testing.T) {
	b := budget.New(0, func() time.Time { return time.Unix(0, 0) })
	if b.Allow() {
		t.Fatal("zero-limit must deny")
	}
}
