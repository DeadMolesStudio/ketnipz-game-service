package game

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/go-park-mail-ru/2018_2_DeadMolesStudio/logger"
)

const (
	MsPerFrame = 20 * time.Millisecond // 50 fps
	GameTime   = 30 * time.Second

	TargetCount        = 4
	TargetVariaty      = 6
	TargetRandomsEvery = 1 * time.Second

	PlayerSpeed         = 1.7
	PlayerBaseY         = 8.7
	PlayerJumpSpeed     = 4
	PlayerGravity       = 0.4
	PlayerWidth         = 10
	PlayerHeight        = 18
	PlayerSuccessPoints = +3
	PlayerFailurePoints = -1

	ProductSpeed  = 0.40
	ProductMinY   = -10 // for better fade out
	ProductWidth  = 5
	ProductHeight = 5
)

//easyjson:json
type PlayerData struct {
	Score      int     `json:"score"`
	X          float64 `json:"percentsX"`  // 0-100
	Y          float64 `json:"percentsY"`  // 0-100
	TargetList []int   `json:"targetList"` // 1-6
	speedY     float64 // jump
}

//easyjson:json
type ProductData struct {
	X     float64 `json:"percentsX"` // 0-100
	Y     float64 `json:"percentsY"` // 0-100
	Type  int     `json:"type"`      // 1-6
	speed float64 // speed of product
}

//easyjson:json
type PointsData struct {
	X      float64 `json:"percentsX"` // 0-100
	Y      float64 `json:"percentsY"` // 0-100
	Who    int     `json:"playerNum"` // 1 or 2 (player number who catched)
	Points int     `json:"points"`    // points of catched item (-1, +3)
}

//easyjson:json
type State struct {
	Player1   *PlayerData    `json:"player1"`
	Player2   *PlayerData    `json:"player2"`
	Products  []*ProductData `json:"products,omitempty"`
	Collected []PointsData   `json:"collected,omitempty"`
}

// Actions are move `LEFT`, `RIGHT` or `JUMP`
type Actions int // byte mask

type GameEngine struct {
	Players map[string]int

	Update chan *ProcessActions

	timer      *time.Timer
	ticker     *time.Ticker
	randomizer *time.Ticker
	state      *State
}

// updateState updates game room state (products move, players and products collide,
// products disappear, points appear, etc.).
func (e *GameEngine) updateState() {
	s := e.state
	player1 := s.Player1
	player2 := s.Player2
	s.Collected = s.Collected[:0] // clear points on screen
	for i := len(s.Products) - 1; i >= 0; i-- {
		s.Products[i].Y -= s.Products[i].speed
		p1caught := objectsCollide(s.Products[i], player1)
		p2caught := objectsCollide(s.Products[i], player2)
		if p1caught {
			e.countPoints(s.Products[i], player1, 1)
		}
		if p2caught {
			e.countPoints(s.Products[i], player2, 2)
		}
		// delete if caught or fade out
		if (p1caught || p2caught) || (s.Products[i].Y < ProductMinY) {
			s.Products = append(s.Products[:i], s.Products[i+1:]...)
		}
	}
	if player1.speedY != 0 {
		performJump(player1)
	}
	if player2.speedY != 0 {
		performJump(player2)
	}
	if len(player1.TargetList) == 0 {
		player1.TargetList = generateNewProductList()
	}
	if len(player2.TargetList) == 0 {
		player2.TargetList = generateNewProductList()
	}
}

// randomTarget randoms new target (product) and appends it to the slice of products.
func (e *GameEngine) randomTarget() {
	t := &ProductData{
		X:     rand.Float64()*90 + 5, // [5, 95]
		Y:     100,
		Type:  rand.Intn(TargetVariaty) + 1,
		speed: ProductSpeed + rand.Float64()*ProductSpeed/2,
	}
	logger.Infof("new product is %v", t)
	e.state.Products = append(e.state.Products, t)
}

// doAction updates player's position: moves him left, right or performs jump.
func (e *GameEngine) doAction(a *ProcessActions) {
	uGameID := a.From
	playerNumber := e.Players[uGameID]
	var player *PlayerData
	if playerNumber == 1 {
		player = e.state.Player1
	} else {
		player = e.state.Player2
	}
	switch a.Actions {
	case 1:
		logger.Debugf("the hero %v moves right", uGameID)
		player.X = math.Min(100, player.X+PlayerSpeed)
	case 10, 111:
		logger.Debugf("the hero %v jumps", uGameID)
		if player.speedY == 0 {
			player.speedY = PlayerJumpSpeed
		}
	case 100:
		logger.Debugf("the hero %v moves left", uGameID)
		player.X = math.Max(0, player.X-PlayerSpeed)
	case 11:
		logger.Debugf("the hero %v moves right and jumps", uGameID)
		if player.speedY == 0 {
			player.speedY = PlayerJumpSpeed
		}
		player.X = math.Min(100, player.X+PlayerSpeed)
	case 110:
		logger.Debugf("the hero %v moves left and jumps", uGameID)
		if player.speedY == 0 {
			player.speedY = PlayerJumpSpeed
		}
		player.X = math.Max(0, player.X-PlayerSpeed)
	case 0, 101: // should not be sent from front-end
		logger.Debugf("the hero %v stands still, nothing to do", uGameID)
	default:
		logger.Errorf("unknown mask from %v: %v", uGameID, a.Actions)
	}
}

// generateNewProductList returns new target list of random products for player.
func generateNewProductList() []int {
	list := make([]int, 0, TargetCount)
	if TargetVariaty >= TargetCount { // list has only unique items
		variaty := make([]int, 0, TargetVariaty)
		for i := 0; i < TargetVariaty; i++ {
			variaty = append(variaty, i+1)
		}
		for i := 0; i < TargetCount; i++ {
			pos := rand.Intn(len(variaty))
			item := variaty[pos]
			list = append(list, item)
			variaty = append(variaty[:pos], variaty[pos+1:]...)
		}
	} else { // variaty is less than target item count so list has repeatable items
		for i := 0; i < TargetCount; i++ {
			list = append(list, rand.Intn(TargetVariaty)+1) // [1, TargetVariaty]
		}
	}
	return list
}

// objectsCollide checks collision of object and player using their coordinates
// and sizes and returns true if they collide. Has size constants from models (sprites).
func objectsCollide(product *ProductData, player *PlayerData) bool {
	// coordinates of the upper left corner of the product
	productX := product.X - (ProductWidth-1)/2
	productY := product.Y - (ProductHeight-8)/2

	// coordinates of the upper left corner of the player
	playerX := player.X - (PlayerWidth-2)/2
	playerY := player.Y - (PlayerHeight-15)/2

	XColl := false
	YColl := false

	if (productX+ProductWidth-0.5 >= playerX) && (productX <= playerX+PlayerWidth-1) {
		XColl = true
	}
	if (productY+ProductHeight-4 >= playerY) && (productY <= playerY+PlayerHeight-7.5) {
		YColl = true
	}

	return XColl && YColl
}

// performJump moves player in Y dimension and reduces his Y-speed.
func performJump(player *PlayerData) {
	player.Y += player.speedY
	player.speedY -= PlayerGravity
	if player.Y <= PlayerBaseY {
		player.speedY = 0
		player.Y = PlayerBaseY
	}
}

// countPoints checks if the product is in player's target list and adds
// PlayerSuccessPoints to his score and deletes from the list if it is or
// reduces the score by PlayerFailurePoints. Points are displayed at the product location.
func (e *GameEngine) countPoints(caught *ProductData, player *PlayerData, playerNum int) {
	itemIsInList := false
	for i := len(player.TargetList) - 1; i >= 0; i-- {
		if caught.Type == player.TargetList[i] {
			player.Score += PlayerSuccessPoints
			// delete from player's target list
			player.TargetList = append(player.TargetList[:i], player.TargetList[i+1:]...)
			itemIsInList = true
		}
	}
	points := PointsData{
		X:   caught.X,
		Y:   caught.Y,
		Who: playerNum,
	}
	if itemIsInList {
		points.Points = PlayerSuccessPoints
		e.state.Collected = append(e.state.Collected, points)
		logger.Infof("player %v caught necessary product %v at (%v, %v)", playerNum, caught.Type, caught.X, caught.Y)
	} else {
		player.Score += PlayerFailurePoints
		points.Points = PlayerFailurePoints
		e.state.Collected = append(e.state.Collected, points)
		logger.Infof("player %v caught wrong product %v at (%v, %v)", playerNum, caught.Type, caught.X, caught.Y)
	}
}

// NewGameEngine initializes new object of GameEngine with given room and players.
func NewGameEngine(r *Room, p1, p2 *Player) (*GameEngine, error) {
	if p1 == nil || p2 == nil {
		return nil, fmt.Errorf("players' data is not valid")
	}
	ge := &GameEngine{
		Players: make(map[string]int),
		Update:  make(chan *ProcessActions, 100),
		state:   NewInitialState(),
	}

	ge.Players[p1.GameSessionID] = 1
	ge.Players[p2.GameSessionID] = 2

	return ge, nil
}

// NewInitialState returns new state initialized with default values.
func NewInitialState() *State {
	return &State{
		Player1: &PlayerData{
			X:          25,
			Y:          PlayerBaseY,
			TargetList: generateNewProductList(),
		},
		Player2: &PlayerData{
			X:          75,
			Y:          PlayerBaseY,
			TargetList: generateNewProductList(),
		},
		Products:  make([]*ProductData, 0, 16),
		Collected: make([]PointsData, 0, 4),
	}
}
