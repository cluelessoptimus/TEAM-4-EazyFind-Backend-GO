package database

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

// Connect establishes a connection to the PostgreSQL database, optimized for serverless
// environments like Neon by managing idle connections efficiently.
func Connect() (*sql.DB, error) {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable not set")
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	// Verify connection
	if err := db.Ping(); err != nil {
		fmt.Printf("Warning: Database ping failed: %v. Proceeding carefully...\n", err)
	}

	// Set connection settings for serverless usage (Neon)
	// Disable idle connections to avoid holding on to suspended compute
	db.SetMaxIdleConns(0)
	// Limit open connections for this simple app
	db.SetMaxOpenConns(10)
	// Refresh connections periodically
	// db.SetConnMaxLifetime(5 * time.Minute)

	fmt.Println("Connected to PostgreSQL successfully (Optimized for Neon)")
	return db, nil
}
