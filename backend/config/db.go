package config

import (
	"log"
	"os"
	"strings"

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

	// PgBouncer in transaction mode (port 6543) does not support prepared
	// statements. Append simple_protocol mode so pgx/v5 uses plain queries.
	if !strings.Contains(dsn, "default_query_exec_mode") {
		if strings.Contains(dsn, "?") {
			dsn += "&default_query_exec_mode=simple_protocol"
		} else {
			dsn += "?default_query_exec_mode=simple_protocol"
		}
	}

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:      logger.Default.LogMode(logger.Info),
		PrepareStmt: false,
	})
	if err != nil {
		log.Fatal("failed to connect to DB")
	}
	log.Println("DB connected ")
}
