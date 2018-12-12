package game

import (
	"sync"

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
	CloseRoom chan string
}

// Run listens to channel Register (processes User) and CloseRoom (closes room with finished game).
func (g *Game) Run() {
	for {
		select {
		case u := <-g.Register:
			logger.Infof("game got new ws connection, user %v, session_id %v", u.UID, u.SessionID)
			g.processUser(u)
		case rID := <-g.CloseRoom:
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
	r := g.findRoom()
	if r == nil {
		logger.Error("max count of rooms")
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
func (g *Game) findRoom() *Room {
	var r *Room
	g.Rooms.Range(func(k, v interface{}) bool {
		rv := v.(*Room)
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

	if r != nil {
		return r
	}
	if g.Total >= MaxRooms {
		return nil
	}

	r = NewRoom()
	g.TotalM.Lock()
	g.Total++
	metrics.AddRoomToCounter()
	g.TotalM.Unlock()
	g.Rooms.Store(r.ID, r)
	logger.Infof("room %v created, total %v", r.ID, g.Total)

	return r
}

// InitGodGameObject initializes new object of Game.
func InitGodGameObject() *Game {
	g = &Game{
		Rooms:     &sync.Map{},
		TotalM:    &sync.Mutex{},
		Register:  make(chan *User, 1),
		CloseRoom: make(chan string, 1),
	}
	return g
}

// AddPlayer processes User to Game.
func AddPlayer(u *User) {
	g.Register <- u
}
