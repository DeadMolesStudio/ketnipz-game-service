package main

import (
	"github.com/gorilla/websocket"
	
	"github.com/go-park-mail-ru/2018_2_DeadMolesStudio/logger"
)

type Player struct {
	Conn *websocket.Conn

	ID   string
	Room *Room
	Data PlayerData
}

type PlayerData struct {
	Username string
	Lives    int
	Points   int
}

type GotMessage struct {
	State *State `json:"state"`
}

type SentMessage struct {
	State *State `json:"state"`
}

func (p *Player) Listen() {
	m := &GotMessage{}
	err := p.Conn.ReadJSON(m)
	if err != nil {
		if websocket.IsUnexpectedCloseError(err) {
			logger.Infof("player %v was disconnected", p.ID)
			p.Room.Unregister <- p
		}
		logger.Error(err)
	}
	// process message
	p.Room.Broadcast <- m.State
}

func (p *Player) Send(s *State) {
	m := &SentMessage{State: s}
	err := p.Conn.WriteJSON(m)
	if err != nil {
		if websocket.IsUnexpectedCloseError(err) {
			logger.Infof("player %v was disconnected", p.ID)
			p.Room.Unregister <- p
		}
		logger.Error(err)
	}
}
