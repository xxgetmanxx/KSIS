package auth

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"poker-duel/internal/database"
)

func hashPassword(password string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(password)))
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
}

func sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func validateLogin(login string) string {
	if login == "" {
		return "Введите логин"
	}
	if len(login) < 3 {
		return "Логин от 3 символов"
	}
	if len(login) > 20 {
		return "Логин до 20 символов"
	}
	for _, c := range login {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-') {
			return "Недопустимые символы"
		}
	}
	return ""
}

func validatePassword(password string) string {
	if password == "" {
		return "Введите пароль"
	}
	if len(password) < 4 {
		return "Пароль от 4 символов"
	}
	if len(password) > 50 {
		return "Пароль до 50 символов"
	}
	return ""
}

func userExists(login string) bool {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE login = $1)`
	err := database.DB.QueryRow(query, login).Scan(&exists)
	if err != nil {
		return false
	}
	return exists
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "Метод не поддерживается"})
		return
	}

	login := strings.TrimSpace(r.FormValue("login"))
	password := r.FormValue("password")

	if errMsg := validateLogin(login); errMsg != "" {
		sendJSON(w, http.StatusBadRequest, ErrorResponse{Error: errMsg})
		return
	}

	if errMsg := validatePassword(password); errMsg != "" {
		sendJSON(w, http.StatusBadRequest, ErrorResponse{Error: errMsg})
		return
	}

	if userExists(login) {
		sendJSON(w, http.StatusConflict, ErrorResponse{Error: "Логин занят"})
		return
	}

	query := `INSERT INTO users (login, password_hash, balance, trophies) VALUES ($1, $2, 10000, 0)`
	_, err := database.DB.Exec(query, login, hashPassword(password))
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Ошибка создания пользователя"})
		return
	}

	sendJSON(w, http.StatusCreated, SuccessResponse{Success: true})
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "Метод не поддерживается"})
		return
	}

	login := strings.TrimSpace(r.FormValue("login"))
	password := r.FormValue("password")

	if errMsg := validateLogin(login); errMsg != "" {
		sendJSON(w, http.StatusBadRequest, ErrorResponse{Error: errMsg})
		return
	}

	if password == "" {
		sendJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Введите пароль"})
		return
	}

	if !userExists(login) {
		sendJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "Неверный логин или пароль"})
		return
	}

	var storedHash string
	query := `SELECT password_hash FROM users WHERE login = $1`
	err := database.DB.QueryRow(query, login).Scan(&storedHash)

	if err != nil || storedHash != hashPassword(password) {
		sendJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "Неверный логин или пароль"})
		return
	}

	var balance, trophies int
	query = `SELECT balance, trophies FROM users WHERE login = $1`
	err = database.DB.QueryRow(query, login).Scan(&balance, &trophies)
	if err != nil {
		balance = 10000
		trophies = 0
	}

	sendJSON(w, http.StatusOK, SuccessResponse{
		Success: true,
		Data: map[string]interface{}{
			"login":    login,
			"balance":  balance,
			"trophies": trophies,
		},
	})
}

func ProfileHandler(w http.ResponseWriter, r *http.Request) {
	login := r.URL.Query().Get("login")
	if login == "" {
		sendJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Логин не указан"})
		return
	}

	var balance, trophies int
	query := `SELECT balance, trophies FROM users WHERE login = $1`
	err := database.DB.QueryRow(query, login).Scan(&balance, &trophies)
	if err != nil {
		balance = 10000
		trophies = 0
	}

	sendJSON(w, http.StatusOK, SuccessResponse{
		Success: true,
		Data: map[string]interface{}{
			"login":    login,
			"balance":  balance,
			"trophies": trophies,
		},
	})
}

