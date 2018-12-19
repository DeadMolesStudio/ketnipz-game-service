package main

import (
	"flag"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/go-park-mail-ru/2018_2_DeadMolesStudio/database"
	"github.com/go-park-mail-ru/2018_2_DeadMolesStudio/logger"
	"github.com/go-park-mail-ru/2018_2_DeadMolesStudio/middleware"
	"github.com/go-park-mail-ru/2018_2_DeadMolesStudio/session"

	"game/game"
	"game/metrics"
)

func main() {
	dbConnStr := flag.String("db_connstr", "postgres@localhost:5432", "postgresql connection string")
	dbName := flag.String("db_name", "postgres", "database name")
	authConnStr := flag.String("auth_connstr", "localhost:8081", "auth-service connection string")
	flag.Parse()

	l := logger.InitLogger()
	defer func() {
		err := l.Sync()
		if err != nil {
			logger.Errorf("error while syncing log data: %v", err)
		}
	}()

	prometheus.MustRegister(metrics.TotalRooms)

	dm := database.InitDatabaseManager(*dbConnStr, *dbName)
	defer dm.Close()

	sm := session.ConnectSessionManager(*authConnStr)
	defer sm.Close()

	g := game.InitGodGameObject(dm)
	go g.Run()

	http.Handle("/metrics", promhttp.Handler())

	http.HandleFunc("/game/ws", middleware.RecoverMiddleware(middleware.AccessLogMiddleware(
		middleware.CORSMiddleware(middleware.SessionMiddleware(http.HandlerFunc(StartGame), sm)))))

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
