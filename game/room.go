package game

import (
	"context"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	uuid "github.com/satori/go.uuid"

	"github.com/go-park-mail-ru/2018_2_DeadMolesStudio/logger"
)

const (
	MaxPlayers = 2

	MsPerFrame = 20 * time.Millisecond // 50 fps
	GameTime   = 5 * time.Second
)

type Room struct {
	ID      string
	Players *sync.Map
	Total   int
	TotalM  *sync.Mutex
	Ticker  *time.Ticker

	Ctx    context.Context
	cancel func()

	Register chan *Player
	Change   chan *State
	GameOver chan *GameOver

	RoomState *State
	Timer     *time.Timer
}

type State struct {
	Data []PlayerData
}

const (
	Success = iota
	TimeOver
	Disconnected
)

type GameOver struct {
	Reason int
	Info   interface{}
}

func (r *Room) Run() {
	logger.Infof("game started in room %v", r.ID)
	r.Players.Range(func(k, v interface{}) bool {
		player := v.(*Player)
		go player.Listen()
		return true
	})
	go r.listenForStateChanges()
	r.Ticker = time.NewTicker(MsPerFrame)
	r.Timer = time.NewTimer(GameTime)
	logger.Infof("timer (%v) started in room %v", GameTime, r.ID)
	for {
		select {
		case <-r.Ticker.C:
			logger.Debugf("room %v tick", r.ID)
			r.broadcast(&SentMessage{
				Status:  "state",
				Payload: r.RoomState,
			})
		case <-r.Timer.C:
			// TODO: move to game engine
			logger.Infof("timer in room %v: time out", r.ID)
			r.finish(&GameOver{
				Reason: TimeOver,
			})
			return
		case res := <-r.GameOver:
			r.finish(res)
			return
		}
	}
}

func (r *Room) listenForStateChanges() {
	for {
		select {
		case s := <-r.Change:
			logger.Infof("got game state %v", s)
			r.RoomState = s
		case <-r.Ctx.Done():
			// <-r.GameOver
			logger.Debugf("killed listen at room %v", r.ID)
			return
		}
	}
}

func (r *Room) broadcast(m *SentMessage) {
	r.Players.Range(func(k, v interface{}) bool {
		player := v.(*Player)
		player.SendMessage <- m
		return true
	})
}

func (r *Room) finish(res *GameOver) {
	r.Ticker.Stop()
	r.Timer.Stop()
	switch res.Reason {
	case Success:
		logger.Infof("room %v: game over with success", r.ID)
		r.broadcast(&SentMessage{
			Status: "game_over",
		})
	case TimeOver:
		logger.Infof("room %v: game over with time over", r.ID)
		r.broadcast(&SentMessage{
			Status: "time_over",
		})
	case Disconnected:
		left := res.Info.(*Player)
		r.Players.Delete(left.GameSessionID)
		logger.Infof("room %v: game over with disconnection of player %v (game session %v)", r.ID, left.UserInfo.UID, left.GameSessionID)
		r.broadcast(&SentMessage{
			Status: "disconnected",
		})
	}

	time.Sleep(1 * time.Second)
	r.cancel()

	r.Players.Range(func(k, v interface{}) bool {
		player := v.(*Player)
		// graceful disconnect
		player.UserInfo.Conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		time.Sleep(1 * time.Second)
		player.UserInfo.Conn.Close()
		logger.Infof("room %v: server disconnected player %v (game session %v)", r.ID, player.UserInfo.UID, player.GameSessionID)
		return true
	})
	logger.Infof("stopped room %v", r.ID)
	g.CloseRoom <- r.ID
}

func NewRoom() *Room {
	ctx, cancel := context.WithCancel(context.Background())
	return &Room{
		ID:       uuid.NewV4().String(),
		Players:  &sync.Map{},
		TotalM:   &sync.Mutex{},
		Ctx:      ctx,
		cancel:   cancel,
		Register: make(chan *Player),
		Change:   make(chan *State),
		GameOver: make(chan *GameOver),
	}
}
