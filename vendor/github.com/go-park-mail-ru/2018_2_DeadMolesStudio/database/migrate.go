package database

import (
	"github.com/rubenv/sql-migrate" // applies migrations

	"github.com/go-park-mail-ru/2018_2_DeadMolesStudio/logger"
)

func (dm *DatabaseManager) makeMigrations() error {
	if dm.db == nil {
		return ErrConnRefused
	}

	migrations := &migrate.FileMigrationSource{
		Dir: "migrations",
	}

	n, err := migrate.Exec(dm.db.DB, "postgres", migrations, migrate.Up)
	if err != nil {
		logger.Error(err)
	} else if n != 0 {
		logger.Infof("Applied %d migrations!", n)
	}

	return err
}
