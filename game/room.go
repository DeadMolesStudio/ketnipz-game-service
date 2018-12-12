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
	ticker  *time.Ticker

	Ctx    context.Context
	cancel func()

	Register chan *Player
	Change   chan *State
	GameOver chan *GameOver

	engine    *GameEngine
	RoomState *State
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
	OpponentID uint       `json:"opponentId"`
	PlayerNum  uint       `json:"playerNum"`
	Constants  *GameConst `json:"stateConst"`
}

//easyjson:json
type GameConst struct {
	GameTime time.Duration `json:"gameTime"`
}

type GameOver struct {
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
	r.engine, err = NewGameEngine(r, player1, player2)
	if err != nil {
		logger.Errorf("engine cannot be created: %v", err)
		return
	}

	player1.SendMessage <- &WSMessageToSend{
		Status: "started",
		Payload: &StartInfo{
			OpponentID: player2.UserInfo.UID,
			PlayerNum:  1,
			Constants: &GameConst{
				GameTime: GameTime,
			},
		},
	}
	player2.SendMessage <- &WSMessageToSend{
		Status: "started",
		Payload: &StartInfo{
			OpponentID: player1.UserInfo.UID,
			PlayerNum:  2,
			Constants: &GameConst{
				GameTime: GameTime,
			},
		},
	}
	go r.listenForStateChanges()
	wg := &sync.WaitGroup{} // wait for engine start
	wg.Add(1)
	go r.engine.Run(wg)
	wg.Wait()
	r.ticker = time.NewTicker(MsPerFrame)
	for {
		select {
		case <-r.ticker.C:
			logger.Debugf("room %v tick", r.ID)
			r.broadcast(&WSMessageToSend{
				Status:  "state",
				Payload: r.RoomState,
			})
		case res := <-r.GameOver:
			logger.Infof("got gameover signal in room %v", r.ID)
			r.finish(res)
			return
		}
	}
}

// listenForStateChanges listens to Change channel (changes instance of game state).
func (r *Room) listenForStateChanges() {
	for {
		select {
		case s := <-r.Change:
			logger.Debugf("got game state %v", s)
			r.RoomState = s
		case <-r.Ctx.Done():
			logger.Debugf("killed listen at room %v", r.ID)
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
func (r *Room) finish(res *GameOver) {
	r.ticker.Stop()
	switch res.Reason {
	case TimeOver:
		logger.Infof("room %v: game over with time over", r.ID)
		r.broadcast(&WSMessageToSend{
			Status: "time_over",
		})
	case Disconnected:
		left := res.Info.(*Player)
		r.Players.Delete(left.GameSessionID)
		logger.Infof("room %v: game over with disconnection of player %v (game session %v)", r.ID, left.UserInfo.UID, left.GameSessionID)
		r.broadcast(&WSMessageToSend{
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

// NewRoom initializes new object of Room.
func NewRoom() *Room {
	ctx, cancel := context.WithCancel(context.Background())
	return &Room{
		ID:        uuid.NewV4().String(),
		Players:   &sync.Map{},
		TotalM:    &sync.Mutex{},
		Ctx:       ctx,
		cancel:    cancel,
		Register:  make(chan *Player, 1),
		Change:    make(chan *State, 1),
		GameOver:  make(chan *GameOver, 1),
		RoomState: new(State),
	}
}
