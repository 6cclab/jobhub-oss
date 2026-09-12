.PHONY: build run dev migrate backfill docker-build docker-run clean tidy vet

build:
	go build -o bin/jobhub ./cmd/server

run:
	go run ./cmd/server/main.go

dev:
	DATABASE_URL=postgres://jobhub:jobhub@localhost:5432/jobhub?sslmode=disable go run ./cmd/server/main.go

# Migrations run automatically on server startup
migrate: run

backfill:
	go run ./cmd/backfill/main.go

docker-build:
	docker build -t jobhub .

docker-run:
	docker run --rm -p 8080:8080 -v jobhub-data:/data jobhub

clean:
	rm -rf bin/ *.db

tidy:
	go mod tidy

vet:
	go vet ./...
