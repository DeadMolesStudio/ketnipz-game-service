package main

import (
	"net/http"

	"github.com/gorilla/websocket"

	"github.com/go-park-mail-ru/2018_2_DeadMolesStudio/logger"
	"github.com/go-park-mail-ru/2018_2_DeadMolesStudio/middleware"
	"github.com/go-park-mail-ru/2018_2_DeadMolesStudio/session"
)

func main() {
	l := logger.InitLogger()
	defer l.Sync()

	sm := session.ConnectSessionManager()
	defer sm.Close()

	g := InitGodGameObject()
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
// @Router /game/ws [GET]
func StartGame(w http.ResponseWriter, r *http.Request) {
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

	AddPlayer(conn)
}
