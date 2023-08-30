package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	_ "github.com/lib/pq"
)

type MigrationResult struct {
	Database string
	Success  bool
	Error    error
}

func main() {
	// Database connection information
	connectionString := "user=username sslmode=disable"
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		log.Fatal("Failed to connect to PostgreSQL:", err)
	}
	defer db.Close()

	// Fetch the list of databases
	databases, err := fetchDatabases(db)
	if err != nil {
		log.Fatal("Failed to fetch databases:", err)
	}

	// Wait group for coordinating goroutines
	var wg sync.WaitGroup

	// Channel for collecting migration results
	resultsCh := make(chan MigrationResult, len(databases))

	// Get the current directory
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	// Loop through each database
	for _, dbName := range databases {
		wg.Add(1) // Increment the WaitGroup counter for each goroutine

		go func(dbName string) {
			defer wg.Done() // Decrement the WaitGroup counter when the goroutine completes

			// Connect to the database
			db, err := sql.Open("postgres", fmt.Sprintf("user=username dbname=%s sslmode=disable", dbName))
			if err != nil {
				resultsCh <- MigrationResult{Database: dbName, Success: false, Error: err}
				return
			}
			defer db.Close()

			// Read migration script from file
			scriptPath := filepath.Join(dir, "src/migration", "migration_script.sql")
			migrationScript, err := os.ReadFile(scriptPath)
			if err != nil {
				resultsCh <- MigrationResult{Database: dbName, Success: false, Error: err}
				return
			}

			// Execute migration script
			_, err = db.Exec(string(migrationScript))
			if err != nil {
				resultsCh <- MigrationResult{Database: dbName, Success: false, Error: err}
				return
			}

			// If migration succeeded
			resultsCh <- MigrationResult{Database: dbName, Success: true, Error: nil}
		}(dbName)
	}

	// Close the results channel after all goroutines are done
	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	// Collect results
	successfulMigrations := []string{}
	failedMigrations := []string{}
	for result := range resultsCh {
		if result.Success {
			successfulMigrations = append(successfulMigrations, result.Database)
		} else {
			failedMigrations = append(failedMigrations, result.Database)
			log.Printf("Migration failed for %s: %v\n", result.Database, result.Error)
		}
	}

	// Print results
	fmt.Println("Successful migrations:", successfulMigrations)
	fmt.Println("Failed migrations:", failedMigrations)
}

// Fetches the list of databases from PostgreSQL
func fetchDatabases(db *sql.DB) ([]string, error) {
	rows, err := db.Query("SELECT datname FROM pg_database WHERE datistemplate = false")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var databases []string
	for rows.Next() {
		var dbName string
		err := rows.Scan(&dbName)
		if err != nil {
			return nil, err
		}
		databases = append(databases, dbName)
	}

	return databases, nil
}
