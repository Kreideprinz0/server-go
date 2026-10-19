package storage

import (
	"errors"
	"sync"
	"testing"
)

func TestStorage_CRUD(t *testing.T) {
	s := New()

	// List пустой
	if got := s.List(); len(got) != 0 {
		t.Fatalf("expected empty list, got %v", got)
	}

	// Create
	it := s.Create("apple", 100)
	if it.ID != 1 || it.Name != "apple" || it.Price != 100 {
		t.Fatalf("unexpected created item: %+v", it)
	}

	// Get
	got, err := s.Get(1)
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if got != it {
		t.Fatalf("Get mismatch: %+v vs %+v", got, it)
	}

	// Update
	upd, err := s.Update(1, "banana", 200)
	if err != nil {
		t.Fatalf("Update error: %v", err)
	}
	if upd.Name != "banana" || upd.Price != 200 {
		t.Fatalf("unexpected updated item: %+v", upd)
	}

	// Delete
	if err := s.Delete(1); err != nil {
		t.Fatalf("Delete error: %v", err)
	}
	if _, err := s.Get(1); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestStorage_NotFound(t *testing.T) {
	s := New()

	if _, err := s.Get(999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get: expected ErrNotFound, got %v", err)
	}
	if _, err := s.Update(999, "x", 1); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Update: expected ErrNotFound, got %v", err)
	}
	if err := s.Delete(999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Delete: expected ErrNotFound, got %v", err)
	}
}

// Проверка конкурентности через -race
func TestStorage_ConcurrentCreate(t *testing.T) {
	s := New()
	const n = 100

	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			s.Create("x", i)
		}()
	}
	wg.Wait()

	if got := len(s.List()); got != n {
		t.Fatalf("expected %d items, got %d", n, got)
	}
}
