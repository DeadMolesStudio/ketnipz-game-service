package game

import (
	"sync"

	"github.com/go-park-mail-ru/2018_2_DeadMolesStudio/logger"
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
			g.TotalM.Unlock()
			logger.Infof("closed room %v, total %v", rID, g.Total)
		}
	}
}

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
	p.SendMessage <- &SentMessage{
		Status: "connected",
	}
	logger.Infof("player %v (game session %v) joined room %v", p.UserInfo.UID, p.GameSessionID, r.ID)

	if r.Total == MaxPlayers {
		go r.Run()
	}
}

func (g *Game) findRoom() *Room {
	var r *Room
	g.Rooms.Range(func(k, v interface{}) bool {
		rv := v.(*Room)
		if rv.Total < MaxPlayers {
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
	g.TotalM.Unlock()
	g.Rooms.Store(r.ID, r)
	logger.Infof("room %v created, total %v", r.ID, g.Total)

	return r
}

func InitGodGameObject() *Game {
	g = &Game{
		Rooms:     &sync.Map{},
		TotalM:    &sync.Mutex{},
		Register:  make(chan *User),
		CloseRoom: make(chan string),
	}
	return g
}

func AddPlayer(u *User) {
	g.Register <- u
}
