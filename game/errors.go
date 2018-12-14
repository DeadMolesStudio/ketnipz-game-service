package game

import "fmt"

var (
	ErrMaxRooms  = fmt.Errorf("max count of rooms")
	ErrIsPlaying = fmt.Errorf("acc is in game now")
)
