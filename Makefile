.PHONY: test test-clean

test:
	docker compose up -d db
	@for i in $$(seq 1 10); do docker compose exec -T db pg_isready -U app_user -d app_db && break || sleep 1; done
	docker compose exec -T db psql -U app_user -d app_db -c "CREATE TABLE IF NOT EXISTS items (id SERIAL PRIMARY KEY, name TEXT NOT NULL, price INTEGER NOT NULL);"
	DATABASE_URL="postgres://app_user:app_password@localhost:5432/app_db?sslmode=disable" go test -race -v ./...

test-clean:
	docker compose down -v