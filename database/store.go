package database

import (
	db "github.com/go-park-mail-ru/2018_2_DeadMolesStudio/database"
)

func ChangeUserCoinAmount(dm *db.DatabaseManager, uID uint, sum int) error {
	dbo, err := dm.DB()
	if err != nil {
		return err
	}
	_, err = dbo.Exec(`
		UPDATE user_profile
		SET coins = coins + $1
		WHERE user_id = $2`,
		sum, uID,
	)
	if err != nil {
		return err
	}

	return nil
}
