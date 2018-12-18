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
)

type Room struct {
	ID      string
	Players *sync.Map
	Total   int
	TotalM  *sync.Mutex

	Ctx    context.Context
	cancel func()

	Unregister chan *Player

	engine *Engine
}

//easyjson:json
type WSMessageToSend struct {
	Status  string      `json:"status"`
	Payload interface{} `json:"payload,omitempty"`
}

const (
	Success = iota
	TimeOver
	Disconnected
)

//easyjson:json
type StartInfo struct {
	OpponentID uint   `json:"opponentId"`
	PlayerNum  uint   `json:"playerNum"`
	Constants  *Const `json:"stateConst"`
}

//easyjson:json
type Const struct {
	GameTime time.Duration `json:"gameTime"`
}

type Ended struct {
	Reason int
	Info   interface{}
}

// Run runs the game in the room.
func (r *Room) Run() {
	logger.Infof("game started in room %v", r.ID)
	var player1, player2 *Player
	i := 1
	r.Players.Range(func(k, v interface{}) bool {
		player := v.(*Player)
		if i == 1 {
			player1 = player
		} else {
			player2 = player
		}
		i++
		go player.Listen()
		return true
	})
	var err error
	r.engine, err = NewEngine(r, player1, player2)
	if err != nil {
		logger.Errorf("engine cannot be created: %v", err)
		return
	}

	player1.SendMessage <- &WSMessageToSend{
		Status: "started",
		Payload: &StartInfo{
			OpponentID: player2.UserInfo.UID,
			PlayerNum:  1,
			Constants: &Const{
				GameTime: GameTime,
			},
		},
	}
	player2.SendMessage <- &WSMessageToSend{
		Status: "started",
		Payload: &StartInfo{
			OpponentID: player1.UserInfo.UID,
			PlayerNum:  2,
			Constants: &Const{
				GameTime: GameTime,
			},
		},
	}

	// run game engine
	r.engine.ticker = time.NewTicker(MsPerFrame)
	r.engine.randomizer = time.NewTicker(TargetRandomsEvery)
	r.engine.timer = time.NewTimer(GameTime)
	for {
		select {
		case <-r.engine.ticker.C:
			logger.Debugf("room %v tick", r.ID)
			r.broadcast(&WSMessageToSend{
				Status:  "state",
				Payload: r.engine.state.copyState(),
			})
			r.engine.updateState()
		case <-r.engine.randomizer.C:
			logger.Debug("new product incoming")
			r.engine.randomTarget()
		case <-r.engine.timer.C:
			logger.Info("time over in game engine")
			r.finish(&Ended{
				Reason: TimeOver,
			})
			logger.Info("end of game engine")
			return
		case a := <-r.engine.Update:
			r.engine.doAction(a)
		case p := <-r.Unregister:
			logger.Infof("player disconnected signal in room %v", r.ID)
			r.finish(&Ended{
				Reason: Disconnected,
				Info:   p,
			})
			return
		}
	}
}

// broadcast sends the message WSMessageToSend to all players in the room.
func (r *Room) broadcast(m *WSMessageToSend) {
	r.Players.Range(func(k, v interface{}) bool {
		player := v.(*Player)
		player.SendMessage <- m
		return true
	})
}

// finish finishes the game in the room.
func (r *Room) finish(res *Ended) {
	r.engine.ticker.Stop()
	r.engine.randomizer.Stop()
	r.engine.timer.Stop()
	switch res.Reason {
	case TimeOver:
		logger.Infof("room %v: game over with time over", r.ID)
		r.broadcast(&WSMessageToSend{
			Status: "time_over",
		})
	case Disconnected:
		left := res.Info.(*Player)
		r.Players.Delete(left.GameSessionID)
		logger.Infof("room %v: game over with disconnection of player %v (game session %v)",
			r.ID, left.UserInfo.UID, left.GameSessionID)
		r.broadcast(&WSMessageToSend{
			Status: "disconnected",
		})
	}
	r.engine.status = res

	time.Sleep(1 * time.Second)
	r.cancel()

	r.Players.Range(func(k, v interface{}) bool {
		player := v.(*Player)
		// graceful disconnect
		_ = player.UserInfo.Conn.SetWriteDeadline(time.Now().Add(1 * time.Second))
		_ = player.UserInfo.Conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		time.Sleep(1 * time.Second)
		player.UserInfo.Conn.Close()
		logger.Infof("room %v: server disconnected player %v (game session %v)",
			r.ID, player.UserInfo.UID, player.GameSessionID)
		return true
	})
	logger.Infof("stopped room %v", r.ID)
	g.CloseRoom <- r
}

// NewRoom initializes new object of Room.
func NewRoom() *Room {
	ctx, cancel := context.WithCancel(context.Background())
	return &Room{
		ID:         uuid.NewV4().String(),
		Players:    &sync.Map{},
		TotalM:     &sync.Mutex{},
		Ctx:        ctx,
		cancel:     cancel,
		Unregister: make(chan *Player, 1),
	}
}
