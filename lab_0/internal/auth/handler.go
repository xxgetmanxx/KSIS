package auth

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"

	"poker-duel/internal/database"
)

func hashPassword(password string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(password)))
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		return
	}
	login := r.FormValue("login")
	password := r.FormValue("password")

	query := `INSERT INTO users (login, password_hash, balance, trophies) VALUES ($1, $2, 10000, 0)`
	_, err := database.DB.Exec(query, login, hashPassword(password))
	if err != nil {
		http.Error(w, "Conflict", http.StatusConflict)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		return
	}
	login := r.FormValue("login")
	password := r.FormValue("password")

	var storedHash string
	query := `SELECT password_hash FROM users WHERE login = $1`
	err := database.DB.QueryRow(query, login).Scan(&storedHash)

	if err != nil || storedHash != hashPassword(password) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func ProfileHandler(w http.ResponseWriter, r *http.Request) {
	login := r.URL.Query().Get("login")
	var balance, trophies int
	query := `SELECT balance, trophies FROM users WHERE login = $1`
	err := database.DB.QueryRow(query, login).Scan(&balance, &trophies)
	if err != nil {
		// Дефолтные значения для демонстрации, если пользователя нет в таблице
		balance = 25000
		trophies = 150
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"login": login, "balance": balance, "trophies": trophies,
	})
}
