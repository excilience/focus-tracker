DB_URL=postgres://focus:focus@localhost:5433/focus_tracker?sslmode=disable

db-up:
	docker compose up -d

db-down:
	docker compose down

db-reset:
	docker compose down -v
	docker compose up -d
	migrate -path migrations -database "$(DB_URL)" up

migrate-up:
	migrate -path migrations -database "$(DB_URL)" up

migrate-down:
	migrate -path migrations -database "$(DB_URL)" down 1

test:
	TEST_DATABASE_URL="$(DB_URL)" go test -v ./...

run-api:
	DATABASE_URL="$(DB_URL)" go run . api