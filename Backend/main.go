package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// 1. Retrieve the database connection string from the environment variable.
	// Example format: postgres://user:password@localhost:5432/dbname
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("Error: DATABASE_URL environment variable is not set")
	}

	// 2. Parse the connection string into a configuration object.
	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		log.Fatalf("Unable to parse database configuration: %v\n", err)
	}

	// 3. Optional: Configure the connection pool settings.
	// For example, you can adjust connection limits, idle timeouts, etc.
	// config.MaxConns = 10
	// config.MinConns = 2

	// 4. Initialize the connection pool.
	// We use context.Background() as the base context.
	ctx := context.Background()
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		log.Fatalf("Unable to create connection pool: %v\n", err)
	}

	// Ensure that the connection pool is properly closed when the application shuts down.
	defer pool.Close()

	// 5. Verify the connection by pinging the database.
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("Unable to ping the database: %v\n", err)
	}

	fmt.Println("Successfully initialized a secure connection pool to PostgreSQL!")

	// Now you can safely use the `pool` object to execute your queries.
	// Example query:
	// var count int
	// err = pool.QueryRow(ctx, "SELECT count(*) FROM my_table").Scan(&count)
}
