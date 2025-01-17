package storage

import (
    "github.com/joho/godotenv"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
    "log"
    "os"
)

type URL struct {
    gorm.Model
    OriginalURL string `gorm:"unique"`
    ShortURL    string `gorm:"unique"`
}

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