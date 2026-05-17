package database

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func InitDB(host, port, user, password, dbname string) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var err error
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("DB Config Error: %v", err)
	}

	if err = DB.Ping(); err != nil {
		log.Fatalf("DB Connection Error: %v. Проверьте запущен ли PostgreSQL.", err)
	}
}
