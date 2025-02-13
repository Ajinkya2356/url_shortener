package storage

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// URL represents the database model for storing shortened URLs.
// It contains the original URL and its corresponding alias.
type URL struct {
	gorm.Model
	// OriginalURL stores the full URL that needs to be shortened
	OriginalURL string `gorm:"unique"`
	// Alias is the shortened version of the URL
	Alias string `gorm:"unique;index"`
}

// InitDB initializes the database connection using environment variables
// and returns a pointer to the database instance.
//
// It loads the DATABASE_URL from .env file and establishes a connection
// to the PostgreSQL database. It also performs auto-migration for the URL model.
//
// Returns:
//   - *gorm.DB: Database instance
//
// Example:
//
//	db := InitDB()
//	// Use db for database operations
func InitDB() *gorm.DB {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Use the DATABASE_URL environment variable
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}

	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database", err)
	}

	db.AutoMigrate(&URL{})
	return db
}
