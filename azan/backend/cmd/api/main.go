package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"azan-backend/internal/command"
	"azan-backend/internal/cqrs"
	"azan-backend/internal/httpapi"
	"azan-backend/internal/query"
	"azan-backend/internal/storage"
)

func main() {
	ctx := context.Background()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://azan:azan@localhost:5432/azan?sslmode=disable"
	}

	store, err := storage.New(ctx, databaseURL)
	if err != nil {
		log.Fatalf("storage: %v", err)
	}
	defer store.Close()

	if err := store.Migrate(ctx); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	commandBus := cqrs.NewCommandBus()
	queryBus := cqrs.NewQueryBus()
	command.RegisterHandlers(commandBus, store)
	query.RegisterHandlers(queryBus, store)

	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":8081"
	}

	server := httpapi.New(commandBus, queryBus)
	log.Printf("azan API listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, server.Router()))
}
