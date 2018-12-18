package game

import (
	"time"

	"github.com/gorilla/websocket"
	uuid "github.com/satori/go.uuid"

	"github.com/go-park-mail-ru/2018_2_DeadMolesStudio/logger"
)

type User struct {
	SessionID string
	UID       uint
	Conn      *websocket.Conn
}

type Player struct {
	UserInfo *User

	GameSessionID string
	Room          *Room

	SendMessage chan *WSMessageToSend
}

//easyjson:json
// GotMessage is a message from client with hero Action: move `LEFT`, `RIGHT` or `JUMP`.
type GotMessage struct {
	Actions Actions `json:"actions"`
}

type ProcessActions struct {
	From string
	Actions
}

// Listen reads messages from player and breaks the loop when player disconnects or game in room ended.
func (p *Player) Listen() {
	for {
		m := &GotMessage{}
		_, raw, err := p.UserInfo.Conn.ReadMessage()
		if err != nil {
			if p.Room.Ctx.Err() != nil {
				logger.Debugf("killed listen player %v at room %v", p.GameSessionID, p.Room.ID)
				return
			}
			if websocket.IsUnexpectedCloseError(err) {
				logger.Infof("listen: player %v was disconnected (game session %v)", p.UserInfo.UID, p.GameSessionID)
			} else {
				logger.Error(err)
			}
			p.Room.Unregister <- p
			return
		}
		err = m.UnmarshalJSON(raw)
		if err != nil {
			logger.Error(err)
			continue
		}
		logger.Debugf("got correct message with action %v from %v", m.Actions, p.GameSessionID)

		if p.Room.engine != nil {
			p.Room.engine.Update <- &ProcessActions{
				From:    p.GameSessionID,
				Actions: m.Actions,
			}
		}
	}
}

// Send writes every message from SendMessage channel to player and breaks the loop when game in room ends.
func (p *Player) Send() {
	for {
		select {
		case m := <-p.SendMessage:
			j, err := m.MarshalJSON()
			if err != nil {
				logger.Error(err)
				continue
			}
			// kick players with low network
			_ = p.UserInfo.Conn.SetWriteDeadline(time.Now().Add(1 * time.Second))
			err = p.UserInfo.Conn.WriteMessage(websocket.TextMessage, j)
			if err != nil {
				if websocket.IsUnexpectedCloseError(err) {
					logger.Infof("send: player %v was disconnected (game session %v)", p.UserInfo.UID, p.GameSessionID)
				} else {
					logger.Error(err)
				}
				p.Room.Unregister <- p
				return
			}
		case <-p.Room.Ctx.Done():
			logger.Debugf("killed send to player %v at room %v", p.GameSessionID, p.Room.ID)
			return
		}
	}
}

// NewPlayer initializes new object of Player with given User.
func NewPlayer(u *User) *Player {
	return &Player{
		UserInfo:      u,
		GameSessionID: uuid.NewV4().String(),
		SendMessage:   make(chan *WSMessageToSend, 100),
	}
}
