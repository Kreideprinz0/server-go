package storage

import (
	"context"
	"errors"
	"os"
	"testing"
)

// helper для получения тестового хранилища
func newTestStorage(t *testing.T) *Storage {
	t.Helper()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Fatal("DATABASE_URL is not set. Please run tests with DATABASE_URL env var (e.g. export DATABASE_URL=postgres://app_user:app_password@localhost:5432/app_db?sslmode=disable)")
	}

	s, err := New(dbURL)
	if err != nil {
		t.Fatalf("failed to connect to db: %v", err)
	}

	// Накатываем миграции
	if err := s.Migrate(context.Background()); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	// Очищаем таблицу перед каждым тестом (чтобы ID начинались с 1)
	_, err = s.db.ExecContext(context.Background(), "TRUNCATE items RESTART IDENTITY")
	if err != nil {
		t.Fatalf("failed to truncate table: %v", err)
	}

	return s
}

func TestStorage_CRUD(t *testing.T) {
	s := newTestStorage(t)
	defer s.Close()
	ctx := context.Background()

	// List пустой
	items, err := s.List(ctx)
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected empty list, got %v", items)
	}

	// Create
	it, err := s.Create(ctx, "apple", 100)
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if it.ID != 1 || it.Name != "apple" || it.Price != 100 {
		t.Fatalf("unexpected created item: %+v", it)
	}

	// Get
	got, err := s.Get(ctx, 1)
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if got != it {
		t.Fatalf("Get mismatch: %+v vs %+v", got, it)
	}

	// Update
	upd, err := s.Update(ctx, 1, "banana", 200)
	if err != nil {
		t.Fatalf("Update error: %v", err)
	}
	if upd.Name != "banana" || upd.Price != 200 {
		t.Fatalf("unexpected updated item: %+v", upd)
	}

	// Delete
	if err := s.Delete(ctx, 1); err != nil {
		t.Fatalf("Delete error: %v", err)
	}
	if _, err := s.Get(ctx, 1); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestStorage_NotFound(t *testing.T) {
	s := newTestStorage(t)
	defer s.Close()
	ctx := context.Background()

	if _, err := s.Get(ctx, 999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get: expected ErrNotFound, got %v", err)
	}
	if _, err := s.Update(ctx, 999, "x", 1); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Update: expected ErrNotFound, got %v", err)
	}
	if err := s.Delete(ctx, 999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Delete: expected ErrNotFound, got %v", err)
	}
}

// Проверка конкурентности (Race Condition).
// В базе данных по умолчанию пул соединений, так что это проверит, что мы не ловим deadlock.
func TestStorage_ConcurrentCreate(t *testing.T) {
	s := newTestStorage(t)
	defer s.Close()
	ctx := context.Background()

	const n = 100
	errCh := make(chan error, n)

	for i := 0; i < n; i++ {
		go func(val int) {
			_, err := s.Create(ctx, "x", val)
			errCh <- err
		}(i)
	}

	for i := 0; i < n; i++ {
		if err := <-errCh; err != nil {
			t.Errorf("concurrent create failed: %v", err)
		}
	}

	items, _ := s.List(ctx)
	if len(items) != n {
		t.Fatalf("expected %d items, got %d", n, len(items))
	}
}
