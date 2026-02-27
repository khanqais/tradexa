package config

import (
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func ConnectDB() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("Url")
	}
	if dsn == "" {
		dsn = os.Getenv("DB_URL")
	}
	if dsn == "" {
		log.Fatal("database URL not found: set DATABASE_URL (or Url/DB_URL)")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatal("failed to connect to DB")
	}
	log.Println("DB connected ")
	DB = db
}
