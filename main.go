package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	_ "github.com/lib/pq"
)

// Configuration defines the parameters for the migration process.
type Configuration struct {
	DBUsername    string
	MigrationDir  string
	DBConnections []string
}

// MigrationResult holds information about the result of a migration.
type MigrationResult struct {
	Database string
	Success  bool
	Error    error
}

func main() {
	// Define configuration
	config := Configuration{
		DBUsername:    "username",
		MigrationDir:  "src/migration",
		DBConnections: []string{"db1", "db2", "db3"},
	}

	// Perform migrations
	results := migrateDatabases(config)

	// Print results
	printMigrationResults(results)
}

// migrateDatabases performs schema migrations for multiple databases.
func migrateDatabases(config Configuration) []MigrationResult {
	var wg sync.WaitGroup
	resultsCh := make(chan MigrationResult, len(config.DBConnections))

	for _, dbName := range config.DBConnections {
		wg.Add(1)
		go func(dbName string) {
			defer wg.Done()

			// Connect to the database
			db, err := connectToDatabase(config.DBUsername, dbName)
			if err != nil {
				resultsCh <- MigrationResult{Database: dbName, Success: false, Error: err}
				return
			}
			defer db.Close()

			// Read migration script from file
			migrationScript, err := readMigrationScript(config.MigrationDir)
			if err != nil {
				resultsCh <- MigrationResult{Database: dbName, Success: false, Error: err}
				return
			}

			// Execute migration script
			err = executeMigration(db, migrationScript)
			if err != nil {
				resultsCh <- MigrationResult{Database: dbName, Success: false, Error: err}
				return
			}

			// If migration succeeded
			resultsCh <- MigrationResult{Database: dbName, Success: true, Error: nil}
		}(dbName)
	}

	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	var results []MigrationResult
	for result := range resultsCh {
		results = append(results, result)
	}

	return results
}

// connectToDatabase connects to the specified database.
func connectToDatabase(username, dbName string) (*sql.DB, error) {
	connectionString := fmt.Sprintf("user=%s dbname=%s sslmode=disable", username, dbName)
	return sql.Open("postgres", connectionString)
}

// readMigrationScript reads the migration script from the specified directory.
func readMigrationScript(migrationDir string) (string, error) {
	scriptPath := filepath.Join(migrationDir, "migration_script.sql")
	migrationScript, err := os.ReadFile(scriptPath)
	if err != nil {
		return "", err
	}
	return string(migrationScript), nil
}

// executeMigration executes the migration script on the given database.
func executeMigration(db *sql.DB, migrationScript string) error {
	_, err := db.Exec(migrationScript)
	return err
}

// printMigrationResults prints the results of the migration process.
func printMigrationResults(results []MigrationResult) {
	fmt.Println("Migration Results:")
	for _, result := range results {
		successStr := "Success"
		if !result.Success {
			successStr = "Failed"
		}
		fmt.Printf("[%s] Database: %s\n", successStr, result.Database)
		if !result.Success {
			fmt.Printf("Error: %v\n", result.Error)
		}
	}
}
