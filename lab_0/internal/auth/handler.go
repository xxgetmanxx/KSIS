package auth

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
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

func StatsHandler(w http.ResponseWriter, r *http.Request) {
	login := r.URL.Query().Get("login")
	if login == "" {
		sendJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Логин не указан"})
		return
	}

	var totalGames, wins, maxWin int
	var history []map[string]interface{}

	var userID int
	query := `SELECT id FROM users WHERE login = $1`
	err := database.DB.QueryRow(query, login).Scan(&userID)
	if err == nil {
		query = `SELECT COUNT(*) FROM games_history WHERE winner_id = $1 OR loser_id = $1`
		database.DB.QueryRow(query, userID).Scan(&totalGames)

		query = `SELECT COUNT(*) FROM games_history WHERE winner_id = $1`
		database.DB.QueryRow(query, userID).Scan(&wins)

		query = `SELECT COALESCE(MAX(pot), 0) FROM games_history WHERE winner_id = $1`
		database.DB.QueryRow(query, userID).Scan(&maxWin)

		query = `SELECT 
			CASE WHEN winner_id = $1 THEN true ELSE false END as won,
			pot,
			mode,
			played_at
		FROM games_history 
		WHERE winner_id = $1 OR loser_id = $1
		ORDER BY played_at DESC
		LIMIT 10`
		rows, err := database.DB.Query(query, userID)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var won bool
				var pot int
				var mode string
				var playedAt string
				rows.Scan(&won, &pot, &mode, &playedAt)
				history = append(history, map[string]interface{}{
					"won":  won,
					"pot":  pot,
					"mode": mode,
				})
			}
		}
	}

	if totalGames == 0 {
		totalGames = 0
		wins = 0
		maxWin = 0
	}

	var winPercent int
	if totalGames > 0 {
		winPercent = (wins * 100) / totalGames
	} else {
		winPercent = 0
	}

	sendJSON(w, http.StatusOK, SuccessResponse{
		Success: true,
		Data: map[string]interface{}{
			"total_games":  totalGames,
			"win_percent":  winPercent,
			"max_win":      maxWin,
			"history":      history,
		},
	})
}

func SaveGameResultHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "Метод не поддерживается"})
		return
	}

	login := strings.TrimSpace(r.FormValue("login"))
	netAmountStr := r.FormValue("net_amount")
	wonStr := r.FormValue("won")
	potStr := r.FormValue("pot")
	mode := r.FormValue("mode")

	log.Println("SaveGameResultHandler called:", login, netAmountStr, wonStr, potStr, mode)

	if login == "" {
		sendJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Логин не указан"})
		return
	}

	var netAmount, pot int
	var won bool
	var err error

	netAmount, err = strconv.Atoi(netAmountStr)
	if err != nil {
		log.Println("Error parsing net_amount:", err)
		sendJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Некорректный net_amount"})
		return
	}

	pot, err = strconv.Atoi(potStr)
	if err != nil {
		log.Println("Error parsing pot:", err)
		sendJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Некорректный pot"})
		return
	}

	won = wonStr == "true"

	log.Println("Parsed values:", netAmount, pot, won)

	tx, err := database.DB.Begin()
	if err != nil {
		log.Println("Error begin tx:", err)
		sendJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Ошибка базы данных"})
		return
	}
	defer tx.Rollback()

	var userID int
	query := `SELECT id FROM users WHERE login = $1`
	err = tx.QueryRow(query, login).Scan(&userID)
	if err != nil {
		log.Println("Error get user id:", err)
		sendJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Пользователь не найден"})
		return
	}

	var botID int
	query = `SELECT id FROM users WHERE login = 'bot'`
	err = tx.QueryRow(query).Scan(&botID)
	if err != nil {
		query = `INSERT INTO users (login, password_hash) VALUES ('bot', 'bot') RETURNING id`
		err = tx.QueryRow(query).Scan(&botID)
		if err != nil {
			log.Println("Error create bot:", err)
			sendJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Ошибка создания бота"})
			return
		}
	}

	var winnerID, loserID int
	if won {
		winnerID = userID
		loserID = botID
	} else {
		winnerID = botID
		loserID = userID
	}

	query = `INSERT INTO games_history (winner_id, loser_id, pot, mode) VALUES ($1, $2, $3, $4)`
	_, err = tx.Exec(query, winnerID, loserID, pot, mode)
	if err != nil {
		log.Println("Error insert history:", err)
		sendJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Ошибка сохранения истории"})
		return
	}

	query = `UPDATE users SET balance = balance + $1 WHERE id = $2`
	_, err = tx.Exec(query, netAmount, userID)
	if err != nil {
		log.Println("Error update balance:", err)
		sendJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Ошибка обновления баланса"})
		return
	}

	if won && mode == "Турнир" {
		query = `UPDATE users SET trophies = trophies + 1 WHERE id = $1`
		_, err = tx.Exec(query, userID)
		if err != nil {
			log.Println("Error update trophies:", err)
			sendJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Ошибка обновления трофеев"})
			return
		}
	}

	var newBalance, newTrophies int
	query = `SELECT balance, trophies FROM users WHERE id = $1`
	err = tx.QueryRow(query, userID).Scan(&newBalance, &newTrophies)
	if err != nil {
		log.Println("Error get new balance:", err)
		sendJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Ошибка получения данных"})
		return
	}

	err = tx.Commit()
	if err != nil {
		log.Println("Error commit:", err)
		sendJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Ошибка коммита"})
		return
	}

	log.Println("Success! New balance:", newBalance)

	sendJSON(w, http.StatusOK, SuccessResponse{
		Success: true,
		Data: map[string]interface{}{
			"balance":  newBalance,
			"trophies": newTrophies,
		},
	})
}

