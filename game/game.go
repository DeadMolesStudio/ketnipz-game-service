package game

import (
	"game/database"
	"game/models"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	db "github.com/go-park-mail-ru/2018_2_DeadMolesStudio/database"
	"github.com/go-park-mail-ru/2018_2_DeadMolesStudio/logger"

	"game/metrics"
)

const (
	MaxRooms = 100500
)

var g *Game

type Game struct {
	Rooms  *sync.Map
	Total  int
	TotalM *sync.Mutex

	Register  chan *User
	CloseRoom chan *Room

	dm *db.DatabaseManager
}

// Run listens to channel Register (processes User) and CloseRoom (closes room with finished game).
func (g *Game) Run() {
	for {
		select {
		case u := <-g.Register:
			logger.Infof("game got new ws connection, user %v, session_id %v", u.UID, u.SessionID)
			go g.processUser(u)
		case r := <-g.CloseRoom:
			g.saveResults(r)
			rID := r.ID
			g.Rooms.Delete(rID)
			g.TotalM.Lock()
			g.Total--
			metrics.SubtractRoomFromCounter()
			g.TotalM.Unlock()
			logger.Infof("closed room %v, total %v", rID, g.Total)
		}
	}
}

// processUser processes User to Room.
func (g *Game) processUser(u *User) {
	p := NewPlayer(u)
	r, err := g.findRoom(p)
	if err != nil {
		switch err {
		case ErrMaxRooms:
			logger.Error(err)
		case ErrIsPlaying:
			logger.Infof("player with id %v is already playing", u.UID)
			m := &WSMessageToSend{
				Status: "playing",
			}
			j, err := m.MarshalJSON()
			if err != nil {
				logger.Error(err)
			}
			u.Conn.SetWriteDeadline(time.Now().Add(1 * time.Second))
			u.Conn.WriteMessage(websocket.TextMessage, j)
			u.Conn.SetWriteDeadline(time.Now().Add(1 * time.Second))
			u.Conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			time.Sleep(1 * time.Second)
			u.Conn.Close()
		}
		return
	}

	r.Players.Store(p.GameSessionID, p)
	r.TotalM.Lock()
	r.Total++
	r.TotalM.Unlock()
	p.Room = r
	go p.Send()
	p.SendMessage <- &WSMessageToSend{
		Status: "connected",
	}
	logger.Infof("player %v (game session %v) joined room %v", p.UserInfo.UID, p.GameSessionID, r.ID)

	if r.Total == MaxPlayers {
		go r.Run()
	}
}

// findRoom searches for free room or creates new room.
func (g *Game) findRoom(p *Player) (*Room, error) {
	var r *Room
	var err error
	g.Rooms.Range(func(k, v interface{}) bool {
		rv := v.(*Room)
		rv.Players.Range(func(k, v interface{}) bool {
			pv := v.(*Player)
			if pv.UserInfo.UID == p.UserInfo.UID {
				err = ErrIsPlaying
				return false
			}
			return true
		})
		if rv.Total < MaxPlayers {
			// TODO: kick dead players
			// rv.Players.Range(func(k, v interface{}) bool {
			// 	pv := v.(*Player)
			// 	logger.Info(pv.UserInfo.Conn.RemoteAddr(), " ", pv.UserInfo.SessionID)
			// 	if err := pv.UserInfo.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			// 		// if err := pv.UserInfo.Conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(2*time.Second)); err != nil {
			// 		logger.Infof("PING ERROR")
			// 		r.Players.Delete(pv.GameSessionID)
			// 		r.TotalM.Lock()
			// 		r.Total--
			// 		r.TotalM.Unlock()
			// 		logger.Infof("player %v was disconnected (game session %v)", pv.UserInfo.UID, pv.GameSessionID)
			// 	}
			// 	return true
			// })
			r = rv
			return false
		}
		return true
	})

	if err != nil {
		return nil, err
	}
	if r != nil {
		return r, nil
	}
	if g.Total >= MaxRooms {
		return nil, ErrMaxRooms
	}

	r = NewRoom()
	g.TotalM.Lock()
	g.Total++
	metrics.AddRoomToCounter()
	g.TotalM.Unlock()
	g.Rooms.Store(r.ID, r)
	logger.Infof("room %v created, total %v", r.ID, g.Total)

	return r, nil
}

// saveResults saves players' results to database (to their profiles)
func (g *Game) saveResults(r *Room) {
	logger.Infof("saving results of room %v...", r.ID)
	if r.engine.status == nil {
		logger.Errorf("saveResults: nil status in room")
		return
	}
	var player1, player2 *Player
	r.Players.Range(func(k, v interface{}) bool {
		pv := v.(*Player)
		if r.engine.Players[pv.GameSessionID] == 1 {
			player1 = pv
		} else {
			player2 = pv
		}
		return true
	})

	switch r.engine.status.Reason {
	case TimeOver:
		player1Score := r.engine.state.Player1.Score
		player2Score := r.engine.state.Player2.Score
		player1Record := &models.Record{
			UID:    player1.UserInfo.UID,
			Record: player1Score,
		}
		player2Record := &models.Record{
			UID:    player2.UserInfo.UID,
			Record: player2Score,
		}
		switch {
		case (player1Score == player2Score) || (player1Score < 0 && player2Score < 0):
			player1Record.GameResult = models.Draw
			player2Record.GameResult = models.Draw
			err := database.UpdateStats(g.dm, player1Record)
			if err != nil {
				logger.Errorf("failed to save player1 %v result: %v", player1.GameSessionID, err)
			}
			err = database.UpdateStats(g.dm, player2Record)
			if err != nil {
				logger.Errorf("failed to save player2 %v result: %v", player2.GameSessionID, err)
			}
		case player1Score > player2Score:
			player1Record.GameResult = models.Win
			player2Record.GameResult = models.Loss
			err := database.UpdateStats(g.dm, player1Record)
			if err != nil {
				logger.Errorf("failed to save player1 %v result: %v", player1.GameSessionID, err)
			}
			err = database.UpdateStats(g.dm, player2Record)
			if err != nil {
				logger.Errorf("failed to save player2 %v result: %v", player2.GameSessionID, err)
			}
		case player1Score < player2Score:
			player2Record.GameResult = models.Win
			player1Record.GameResult = models.Loss
			err := database.UpdateStats(g.dm, player2Record)
			if err != nil {
				logger.Errorf("failed to save player2 %v result: %v", player2.GameSessionID, err)
			}
			err = database.UpdateStats(g.dm, player1Record)
			if err != nil {
				logger.Errorf("failed to save player1 %v result: %v", player1.GameSessionID, err)
			}
		}
	case Disconnected:
		left := r.engine.status.Info.(*Player)
		switch {
		case player1 != nil && player1.GameSessionID != left.GameSessionID:
			// player 1 is winner
			err := database.UpdateStats(g.dm, &models.Record{
				UID:        player1.UserInfo.UID,
				Record:     r.engine.state.Player1.Score,
				GameResult: models.Win,
			})
			if err != nil {
				logger.Errorf("failed to save player1 %v result: %v", player1.GameSessionID, err)
			}
			// player 2 is loser (left game)
			err = database.UpdateStats(g.dm, &models.Record{
				UID:        left.UserInfo.UID,
				Record:     r.engine.state.Player2.Score,
				GameResult: models.Loss,
			})
			if err != nil {
				logger.Errorf("failed to save left player2 %v result: %v", left.GameSessionID, err)
			}
		case player2 != nil && player2.GameSessionID != left.GameSessionID:
			// player 2 is winner
			err := database.UpdateStats(g.dm, &models.Record{
				UID:        player2.UserInfo.UID,
				Record:     r.engine.state.Player2.Score,
				GameResult: models.Win,
			})
			if err != nil {
				logger.Errorf("failed to save player2 %v result: %v", player2.GameSessionID, err)
			}
			// player 1 is loser (left game)
			err = database.UpdateStats(g.dm, &models.Record{
				UID:        left.UserInfo.UID,
				Record:     r.engine.state.Player1.Score,
				GameResult: models.Loss,
			})
			if err != nil {
				logger.Errorf("failed to save left player1 %v result: %v", left.GameSessionID, err)
			}
		default:
			logger.Error("invalid data about left player and winner")
		}
	}
}

// InitGodGameObject initializes new object of Game.
func InitGodGameObject(dm *db.DatabaseManager) *Game {
	g = &Game{
		Rooms:     &sync.Map{},
		TotalM:    &sync.Mutex{},
		Register:  make(chan *User, 1),
		CloseRoom: make(chan *Room, 1),
		dm:        dm,
	}
	return g
}

// AddPlayer processes User to Game.
func AddPlayer(u *User) {
	g.Register <- u
}
