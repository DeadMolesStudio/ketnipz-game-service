package database

import (
	"fmt"

	db "github.com/go-park-mail-ru/2018_2_DeadMolesStudio/database"

	"game/models"
)

func UpdateStats(dm *db.DatabaseManager, r *models.Record) error {
	dbo, err := dm.DB()
	if err != nil {
		return err
	}
	q := `
		UPDATE user_profile
		SET record = GREATEST($1, record), `
	switch r.GameResult {
	case models.Win:
		q += "win = win + 1"
	case models.Draw:
		q += "draws = draws + 1"
	case models.Loss:
		q += "loss = loss + 1"
	default:
		return fmt.Errorf("unknown GameResult value in UpdateStats")
	}
	q += `
		WHERE user_id = $2`
	_, err = dbo.Exec(q, r.Record, r.UID)
	if err != nil {
		return err
	}

	return nil
}
