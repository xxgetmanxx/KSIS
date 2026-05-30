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

	createTables()
}

func createTables() {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			login VARCHAR(50) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			balance INT DEFAULT 10000,
			trophies INT DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS games_history (
			id SERIAL PRIMARY KEY,
			winner_id INT REFERENCES users(id),
			loser_id INT REFERENCES users(id),
			pot INT NOT NULL,
			net_amount INT DEFAULT 0,
			mode VARCHAR(50),
			round INT DEFAULT 0,
			played_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`ALTER TABLE games_history ADD COLUMN IF NOT EXISTS round INT DEFAULT 0`,
		`ALTER TABLE games_history ADD COLUMN IF NOT EXISTS net_amount INT DEFAULT 0`,
	}

	for _, q := range queries {
		_, err := DB.Exec(q)
		if err != nil {
			log.Printf("Warning: %v", err)
		}
	}
}
