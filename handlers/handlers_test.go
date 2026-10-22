package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"server-go/storage"
)

func newTestServer(t *testing.T) (*httptest.Server, *storage.Storage) {
	t.Helper()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Fatal("DATABASE_URL is not set")
	}

	store, err := storage.New(dbURL)
	if err != nil {
		t.Fatalf("failed to connect to db: %v", err)
	}

	// Очищаем базу перед тестом
	_, err = store.ExecRaw("TRUNCATE items RESTART IDENTITY")
	if err != nil {
		t.Fatalf("truncate items %v", err)
	}

	h := New(store)
	mux := http.NewServeMux()
	h.Routes(mux)

	srv := httptest.NewServer(mux)
	return srv, store
}

func doJSON(t *testing.T, method, url string, body any) (*http.Response, []byte) {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode: %v", err)
		}
	}
	req, err := http.NewRequest(method, url, &buf)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()

	var out bytes.Buffer
	if _, err := out.ReadFrom(resp.Body); err != nil {
		t.Fatalf("read body: %v", err)
	}
	return resp, out.Bytes()
}

func TestAPI_FullCRUDCycle(t *testing.T) {
	srv, store := newTestServer(t)
	defer srv.Close()
	defer store.Close()

	// 1. Пустой список
	resp, body := doJSON(t, http.MethodGet, srv.URL+"/items", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list status: %d body=%s", resp.StatusCode, body)
	}
	if string(body) != "[]\n" {
		t.Fatalf("expected empty array, got %q", string(body))
	}

	// 2. Create
	resp, body = doJSON(t, http.MethodPost, srv.URL+"/items", map[string]any{"name": "apple", "price": 100})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create status: %d body=%s", resp.StatusCode, body)
	}
	var created storage.Item
	if err := json.Unmarshal(body, &created); err != nil {
		t.Fatalf("decode created: %v", err)
	}
	if created.ID != 1 || created.Name != "apple" || created.Price != 100 {
		t.Fatalf("unexpected created: %+v", created)
	}

	// 3. Get
	resp, body = doJSON(t, http.MethodGet, srv.URL+"/items/1", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get status: %d body=%s", resp.StatusCode, body)
	}

	// 4. Update
	resp, body = doJSON(t, http.MethodPut, srv.URL+"/items/1", map[string]any{"name": "banana", "price": 200})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update status: %d body=%s", resp.StatusCode, body)
	}
	var updated storage.Item
	_ = json.Unmarshal(body, &updated)
	if updated.Name != "banana" || updated.Price != 200 {
		t.Fatalf("unexpected updated: %+v", updated)
	}

	// 5. Delete
	resp, body = doJSON(t, http.MethodDelete, srv.URL+"/items/1", nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status: %d body=%s", resp.StatusCode, body)
	}

	// 6. Get после удаления -> 404
	resp, body = doJSON(t, http.MethodGet, srv.URL+"/items/1", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", resp.StatusCode, body)
	}
}

func TestAPI_CreateValidation(t *testing.T) {
	srv, store := newTestServer(t)
	defer srv.Close()
	defer store.Close()

	cases := []struct {
		name string
		body any
		want int
	}{
		{"empty name", map[string]any{"name": "", "price": 1}, http.StatusBadRequest},
		{"invalid json handled below", nil, http.StatusBadRequest}, // nil -> пустое тело
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, body := doJSON(t, http.MethodPost, srv.URL+"/items", tc.body)
			if resp.StatusCode != tc.want {
				t.Fatalf("want %d got %d body=%s", tc.want, resp.StatusCode, body)
			}
		})
	}
}

func TestAPI_InvalidID(t *testing.T) {
	srv, store := newTestServer(t)
	defer srv.Close()
	defer store.Close()

	resp, body := doJSON(t, http.MethodGet, srv.URL+"/items/abc", nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", resp.StatusCode, body)
	}
}
