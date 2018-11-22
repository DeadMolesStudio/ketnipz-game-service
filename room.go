package main

import (
	"time"

	"github.com/satori/go.uuid"

	"github.com/go-park-mail-ru/2018_2_DeadMolesStudio/logger"
)

const (
	MaxPlayers = 2
)

type Room struct {
	ID      string
	Players map[string]*Player
	Ticker  *time.Ticker

	Register   chan *Player
	Unregister chan *Player
	Broadcast  chan *State

	RoomState *State
}

type State struct {
	Data []PlayerData
}

func NewRoom() *Room {
	id := uuid.NewV4().String()
	return &Room{
		ID:         id,
		Players:    make(map[string]*Player),
		Register:   make(chan *Player),
		Unregister: make(chan *Player),
		Broadcast:  make(chan *State),
	}
}

func (r *Room) Run() {
	r.Ticker = time.NewTicker(time.Second)
	go r.ListenMessages()
	for {
		<-r.Ticker.C

		logger.Infof("room %v tick", r.ID)
	}
}

func (r *Room) ListenMessages() {
	for k := range r.Players {
		go func(p *Player) {
			for {
				p.Listen()
			}
		}(r.Players[k])
	}

	for {
		s := <-r.Broadcast
		// change room state from message
		logger.Info("got game state %v", s)
	}
}
