package main

import (
	"github.com/gorilla/websocket"
	"github.com/satori/go.uuid"

	"github.com/go-park-mail-ru/2018_2_DeadMolesStudio/logger"
)

const (
	MaxRooms = 100500
)

var g *Game

type Game struct {
	Rooms map[string]*Room

	Register chan *websocket.Conn
}

func (g *Game) Run() {
	for {
		conn := <-g.Register
		logger.Info("game got new ws connection")
		g.processConn(conn)
	}
}

func (g *Game) findRoom() *Room {
	for _, v := range g.Rooms {
		if len(v.Players) < MaxPlayers {
			return v
		}
	}

	if len(g.Rooms) >= MaxRooms {
		return nil
	}

	r := NewRoom()
	g.Rooms[r.ID] = r
	logger.Infof("room %v created", r.ID)

	return r
}

func (g *Game) processConn(conn *websocket.Conn) {
	id := uuid.NewV4().String()
	p := &Player{
		Conn: conn,
		ID:   id,
	}
	r := g.findRoom()
	if r == nil {
		return
	}
	r.Players[p.ID] = p
	p.Room = r
	logger.Infof("player %v joined room %v", p.ID, r.ID)

	if len(r.Players) == MaxPlayers {
		go r.Run()
	}
}

func NewGame() *Game {
	return &Game{
		Rooms:    make(map[string]*Room),
		Register: make(chan *websocket.Conn),
	}
}

func InitGodGameObject() *Game {
	g = NewGame()
	return g
}

func AddPlayer(conn *websocket.Conn) {
	g.Register <- conn
}
