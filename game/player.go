package game

import (
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
	Data          PlayerData

	Disconnect  chan struct{}
	SendMessage chan *SentMessage
}

type PlayerData struct {
	Username string
	Lives    int
	Points   int
}

//easyjson:json
// GotMessage is a message from client with hero action: move `LEFT`, `RIGHT` or `JUMP`
type GotMessage struct {
	Action string `json:"action"`
}

//easyjson:json
type SentMessage struct {
	Status  string      `json:"status"`
	Payload interface{} `json:"payload,omitempty"`
}

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
				logger.Infof("player %v was disconnected (game session %v)", p.UserInfo.UID, p.GameSessionID)
			} else {
				logger.Error(err)
			}
			p.Room.GameOver <- &GameOver{
				Reason: Disconnected,
				Info:   p,
			}
			return
		}
		err = m.UnmarshalJSON(raw)
		if err != nil {
			logger.Error(err)
			continue
		}

		// TODO: game engine should validate state
		p.Room.Change <- &State{}
	}
}

func (p *Player) Send() {
	for {
		select {
		case m := <-p.SendMessage:
			j, err := m.MarshalJSON()
			if err != nil {
				logger.Error(err)
				continue
			}
			err = p.UserInfo.Conn.WriteMessage(websocket.BinaryMessage, j)
			if err != nil {
				if websocket.IsUnexpectedCloseError(err) {
					logger.Infof("player %v was disconnected (game session %v)", p.UserInfo.UID, p.GameSessionID)
				} else {
					logger.Error(err)
				}
				p.Room.GameOver <- &GameOver{
					Reason: Disconnected,
					Info:   p,
				}
				return
			}
		case <-p.Room.Ctx.Done():
			logger.Debugf("killed send to player %v at room %v", p.GameSessionID, p.Room.ID)
			return
		}
	}
}

func NewPlayer(u *User) *Player {
	return &Player{
		UserInfo:      u,
		GameSessionID: uuid.NewV4().String(),
		SendMessage:   make(chan *SentMessage, 100),
	}
}
