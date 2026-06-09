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
	Success bool                   `json:"success"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

func sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func tournamentTrophyDelta(won bool, round int) int {
	if round < 1 {
		round = 1
	}
	if won {
		switch round {
		case 3:
			return 4
		case 2:
			return 2
		default:
			return 1
		}
	}
	switch round {
	case 3:
		return -1
	case 2:
		return -2
	default:
		return -4
	}
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "Метод не поддерживается"})
		return
	}

	login := strings.TrimSpace(r.FormValue("login"))
	password := r.FormValue("password")

	if login == "" || password == "" {
		sendJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Введите логин и пароль"})
		return
	}

	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE login = $1)`
	database.DB.QueryRow(query, login).Scan(&exists)

	if exists {
		sendJSON(w, http.StatusConflict, ErrorResponse{Error: "Логин уже занят"})
		return
	}

	passwordHash := hashPassword(password)
	query = `INSERT INTO users (login, password_hash) VALUES ($1, $2)`
	_, err := database.DB.Exec(query, login, passwordHash)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Ошибка базы данных"})
		return
	}

	sendJSON(w, http.StatusOK, SuccessResponse{Success: true})
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "Метод не поддерживается"})
		return
	}

	login := strings.TrimSpace(r.FormValue("login"))
	password := r.FormValue("password")

	if login == "" || password == "" {
		sendJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Введите логин и пароль"})
		return
	}

	var storedHash string
	var balance, trophies int
	query := `SELECT password_hash, balance, trophies FROM users WHERE login = $1`
	err := database.DB.QueryRow(query, login).Scan(&storedHash, &balance, &trophies)
	if err != nil {
		sendJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "Неверный логин или пароль"})
		return
	}

	if hashPassword(password) != storedHash {
		sendJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "Неверный логин или пароль"})
		return
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

		query = `SELECT COALESCE(MAX(net_amount), 0) FROM games_history WHERE winner_id = $1`
		database.DB.QueryRow(query, userID).Scan(&maxWin)

		query = `SELECT 
			CASE WHEN winner_id = $1 THEN true ELSE false END as won,
			pot,
			net_amount,
			mode,
			round,
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
				var netAmount int
				var mode string
				var round int
				var playedAt string
				rows.Scan(&won, &pot, &netAmount, &mode, &round, &playedAt)
				history = append(history, map[string]interface{}{
					"won":        won,
					"pot":        pot,
					"net_amount": netAmount,
					"mode":       mode,
					"round":      round,
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
			"total_games": totalGames,
			"win_percent": winPercent,
			"max_win":     maxWin,
			"history":     history,
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
	roundStr := r.FormValue("round")

	if login == "" {
		sendJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Логин не указан"})
		return
	}

	var netAmount, pot, round int
	netAmount, _ = strconv.Atoi(netAmountStr)
	pot, _ = strconv.Atoi(potStr)
	round, _ = strconv.Atoi(roundStr)
	if round < 1 {
		round = 1
	}
	won := wonStr == "true"

	tx, err := database.DB.Begin()
	if err != nil {
		log.Println("DB Begin error:", err)
		sendJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Ошибка базы данных"})
		return
	}

	var userID int
	query := `SELECT id FROM users WHERE login = $1`
	if err = tx.QueryRow(query, login).Scan(&userID); err != nil {
		log.Println("Get user ID error:", err)
		tx.Rollback()
		sendJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Пользователь не найден"})
		return
	}

	var botID int
	query = `SELECT id FROM users WHERE login = 'bot'`
	if err = tx.QueryRow(query).Scan(&botID); err != nil {
		query = `INSERT INTO users (login, password_hash) VALUES ('bot', 'bot') RETURNING id`
		if err = tx.QueryRow(query).Scan(&botID); err != nil {
			log.Println("Insert bot error:", err)
			tx.Rollback()
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

	query = `INSERT INTO games_history (winner_id, loser_id, pot, mode, round, net_amount) VALUES ($1, $2, $3, $4, $5, $6)`
	if _, err = tx.Exec(query, winnerID, loserID, pot, mode, round, netAmount); err != nil {
		log.Println("Insert history error:", err)
		tx.Rollback()
		sendJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Ошибка сохранения истории"})
		return
	}

	query = `UPDATE users SET balance = balance + $1 WHERE id = $2`
	if _, err = tx.Exec(query, netAmount, userID); err != nil {
		log.Println("Update balance error:", err)
		tx.Rollback()
		sendJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Ошибка обновления баланса"})
		return
	}

	if mode == "Турнир" {
		delta := tournamentTrophyDelta(won, round)
		if delta != 0 {
			query = `UPDATE users SET trophies = GREATEST(trophies + $1, 0) WHERE id = $2`
			if _, err = tx.Exec(query, delta, userID); err != nil {
				log.Println("Update trophies error:", err)
			}
		}
	}

	var newBalance, newTrophies int
	query = `SELECT balance, trophies FROM users WHERE id = $1`
	if err = tx.QueryRow(query, userID).Scan(&newBalance, &newTrophies); err != nil {
		log.Println("Get new balance error:", err)
		tx.Rollback()
		sendJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Ошибка получения данных"})
		return
	}

	if err = tx.Commit(); err != nil {
		log.Println("Commit error:", err)
		sendJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Ошибка коммита"})
		return
	}

	sendJSON(w, http.StatusOK, SuccessResponse{
		Success: true,
		Data: map[string]interface{}{
			"balance":  newBalance,
			"trophies": newTrophies,
		},
	})
}
