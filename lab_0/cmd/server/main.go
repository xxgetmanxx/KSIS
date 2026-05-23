package main

import (
	"log"
	"net/http"

	"poker-duel/internal/auth"
	"poker-duel/internal/database"
	"poker-duel/internal/network"
)

func main() {
	// Инициализация подключения к БД PostgreSQL
	database.InitDB("127.0.0.1", "5432", "postgres", "postgres", "postgres")
	defer database.DB.Close()

	// Инициализация сетевого WebSocket-хаба очередей
	hub := network.NewHub()
	go hub.Run()

	fs := http.FileServer(http.Dir("./public"))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/auth/register":
			auth.RegisterHandler(w, r)
		case "/api/auth/login":
			auth.LoginHandler(w, r)
		case "/api/user/profile":
			auth.ProfileHandler(w, r)
		case "/api/user/stats":
			auth.StatsHandler(w, r)
		case "/api/game/save-result":
			auth.SaveGameResultHandler(w, r)
		case "/ws":
			network.ServeWS(hub, w, r)
		default:
			fs.ServeHTTP(w, r)
		}
	})

	log.Println("Стеклянный покерный сервер успешно запущен на порту :8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Fatal: %v", err)
	}
}
