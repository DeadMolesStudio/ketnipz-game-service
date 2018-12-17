package models

const (
	Win = iota
	Loss
	Draw
)

type Record struct {
	UID uint

	Record     int
	GameResult int
}
