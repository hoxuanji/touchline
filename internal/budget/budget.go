package budget

import (
	"sync"
	"time"
)

type Budget struct {
	mu    sync.Mutex
	limit int
	now   func() time.Time
	day   string
	used  int
}

func New(limit int, now func() time.Time) *Budget {
	return &Budget{limit: limit, now: now}
}

func (b *Budget) rollover() {
	d := b.now().UTC().Format("2006-01-02")
	if d != b.day {
		b.day = d
		b.used = 0
	}
}

func (b *Budget) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.rollover()
	if b.used >= b.limit {
		return false
	}
	b.used++
	return true
}

func (b *Budget) Remaining() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.rollover()
	r := b.limit - b.used
	if r < 0 {
		return 0
	}
	return r
}
