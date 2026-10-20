package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"server-go/handlers"
	"server-go/storage"
)

func main() {
	// Берем строку подключения из переменных окружения (Docker Compose передаст её)
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// Значение по умолчанию для локального запуска (если запускаем go run . без докера)
		dbURL = "postgres://app_user:app_password@localhost:5432/app_db?sslmode=disable"
	}

	// 1. Подключаемся к БД
	store, err := storage.New(dbURL)
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}
	defer store.Close()

	// 2. Применяем миграции (создаем таблицу)
	if err := store.Migrate(context.Background()); err != nil {
		log.Fatalf("failed to migrate db: %v", err)
	}
	log.Println("db connected and migrated")

	// 3. Настраиваем HTTP
	h := handlers.New(store)
	mux := http.NewServeMux()
	h.Routes(mux)

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      logging(mux),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		log.Println("server started on :8080")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	// 4. Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("shutdown error: %v", err)
	}
	log.Println("server stopped")
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}
