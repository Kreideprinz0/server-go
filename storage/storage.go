package storage

import (
	"errors"
	"sync"
)

var (
	ErrNotFound = errors.New("item not found")
)

// Item - наша сущность
type Item struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Price int    `json:"price"`
}

// Storage - потокобезопасное in-memory хранилище
type Storage struct {
	mu     sync.RWMutex
	items  map[int]Item
	nextID int
}

func New() *Storage {
	return &Storage{
		items:  make(map[int]Item),
		nextID: 1,
	}
}

// List - возвращает все элементы
func (s *Storage) List() []Item {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]Item, 0, len(s.items))
	for _, it := range s.items {
		result = append(result, it)
	}
	return result
}

// Get - возвращает элемент по ID
func (s *Storage) Get(id int) (Item, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	it, ok := s.items[id]
	if !ok {
		return Item{}, ErrNotFound
	}
	return it, nil
}

// Create - создаёт новый элемент и возвращает его с присвоенным ID
func (s *Storage) Create(name string, price int) Item {
	s.mu.Lock()
	defer s.mu.Unlock()

	it := Item{
		ID:    s.nextID,
		Name:  name,
		Price: price,
	}
	s.items[it.ID] = it
	s.nextID++
	return it
}

// Update - обновляет элемент
func (s *Storage) Update(id int, name string, price int) (Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.items[id]; !ok {
		return Item{}, ErrNotFound
	}

	it := Item{ID: id, Name: name, Price: price}
	s.items[id] = it
	return it, nil
}

// Delete - удаляет элемент
func (s *Storage) Delete(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.items[id]; !ok {
		return ErrNotFound
	}
	delete(s.items, id)
	return nil
}
