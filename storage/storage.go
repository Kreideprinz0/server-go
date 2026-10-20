package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var (
	ErrNotFound = errors.New("item not found")
)

// Item
type Item struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Price int    `json:"price"`
}

// Storage - хранилище на базе PostgreSQL
type Storage struct {
	db *sql.DB
}

// New - подключается к базе данных
func New(connStr string) (*Storage, error) {
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	return &Storage{db: db}, nil
}

// Close - закрывает пул соединений с БД
func (s *Storage) Close() error {
	return s.db.Close()
}

// Migrate - создает таблицу при первом запуске
func (s *Storage) Migrate(ctx context.Context) error {
	query := `
	CREATE TABLE IF NOT EXISTS items (
		id SERIAL PRIMARY KEY,
		name TEXT NOT NULL,
		price INTEGER NOT NULL
	);`
	_, err := s.db.ExecContext(ctx, query)
	return err
}

// List - возвращает все элементы
func (s *Storage) List(ctx context.Context) ([]Item, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id, name, price FROM items ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Item
	for rows.Next() {
		var it Item
		if err := rows.Scan(&it.ID, &it.Name, &it.Price); err != nil {
			return nil, err
		}
		result = append(result, it)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if result == nil {
		result = []Item{}
	}
	return result, nil
}

// Get - возвращает элемент по ID
func (s *Storage) Get(ctx context.Context, id int) (Item, error) {
	var it Item
	err := s.db.QueryRowContext(ctx, "SELECT id, name, price FROM items WHERE id = $1", id).
		Scan(&it.ID, &it.Name, &it.Price)

	if errors.Is(err, sql.ErrNoRows) {
		return Item{}, ErrNotFound
	}
	if err != nil {
		return Item{}, err
	}
	return it, nil
}

// Create - создаёт новый элемент и возвращает его с присвоенным ID
func (s *Storage) Create(ctx context.Context, name string, price int) (Item, error) {
	var it Item
	err := s.db.QueryRowContext(ctx,
		"INSERT INTO items (name, price) VALUES ($1, $2) RETURNING id, name, price",
		name, price,
	).Scan(&it.ID, &it.Name, &it.Price)

	if err != nil {
		return Item{}, err
	}
	return it, nil
}

// Update - обновляет элемент
func (s *Storage) Update(ctx context.Context, id int, name string, price int) (Item, error) {
	var it Item
	err := s.db.QueryRowContext(ctx,
		"UPDATE items SET name = $1, price = $2 WHERE id = $3 RETURNING id, name, price",
		name, price, id,
	).Scan(&it.ID, &it.Name, &it.Price)

	if errors.Is(err, sql.ErrNoRows) {
		return Item{}, ErrNotFound
	}
	if err != nil {
		return Item{}, err
	}
	return it, nil
}

// Delete - удаляет элемент
func (s *Storage) Delete(ctx context.Context, id int) error {
	res, err := s.db.ExecContext(ctx, "DELETE FROM items WHERE id = $1", id)
	if err != nil {
		return err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

// ExecRaw - чистит бд (ТОЛЬКО ДЛЯ ТЕСТОВ)
func (s *Storage) ExecRaw(query string) (sql.Result, error) {
	return s.db.Exec(query)
}
