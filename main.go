package main

import (
	"net/http"

	"github.com/gorilla/websocket"

	"github.com/go-park-mail-ru/2018_2_DeadMolesStudio/database"
	"github.com/go-park-mail-ru/2018_2_DeadMolesStudio/logger"
	"github.com/go-park-mail-ru/2018_2_DeadMolesStudio/middleware"
	"github.com/go-park-mail-ru/2018_2_DeadMolesStudio/session"

	"game/game"
)

func main() {
	l := logger.InitLogger()
	defer l.Sync()

	db := database.InitDB("postgres@postgres:5432", "ketnipz")
	defer db.Close()

	sm := session.ConnectSessionManager()
	defer sm.Close()

	g := game.InitGodGameObject()
	go g.Run()

	http.HandleFunc("/game/ws", middleware.RecoverMiddleware(middleware.AccessLogMiddleware(
		middleware.CORSMiddleware(middleware.SessionMiddleware(StartGame)))))

	logger.Info("starting server at: ", 8082)
	logger.Panic(http.ListenAndServe(":8082", nil))
}

// @Summary Начать игру по WebSocket
// @Description Инициализирует соединение для пользователя
// @ID get-game-ws
// @Success 101 "Switching Protocols"
// @Failure 400 "Нет нужных заголовков"
// @Failure 401 "Не вошел"
// @Router /game/ws [GET]
func StartGame(w http.ResponseWriter, r *http.Request) {
	u := &game.User{}

	if r.Context().Value(middleware.KeyIsAuthenticated).(bool) {
		u.SessionID = r.Context().Value(middleware.KeySessionID).(string)
		u.UID = r.Context().Value(middleware.KeyUserID).(uint)
	} else {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("Cannot upgrade connection: ", err)
		return
	}
	u.Conn = conn

	game.AddPlayer(u)
}
