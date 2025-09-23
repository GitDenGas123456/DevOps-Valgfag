package database

import (
	"database/sql"
	_"modernc.org/sqlite"
	"log"
)

var DB *sql.DB

// InitDB initializes the SQLite database connection.
func InitDB(filepath string) {
	var err error
	DB, err = sql.Open("sqlite", filepath)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}

	if err = DB.Ping(); err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}

	log.Println("Connected to database:", filepath)
}
