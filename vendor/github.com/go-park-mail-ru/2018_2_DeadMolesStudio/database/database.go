package database

import (
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // postgres driver

	"github.com/go-park-mail-ru/2018_2_DeadMolesStudio/logger"
)

// nolint: golint
type DatabaseManager struct {
	db *sqlx.DB
}

func InitDatabaseManager(address, database string) *DatabaseManager {
	var err error
	dm := &DatabaseManager{}
	dm.db, err = sqlx.Open("postgres",
		"postgres://"+address+"/"+database+"?sslmode=disable")
	if err != nil {
		logger.Panic(err)
	}

	if err := dm.db.Ping(); err != nil {
		logger.Panic(err)
	}

	logger.Infof("Successfully connected to %v, database %v", address, database)

	_ = dm.makeMigrations()

	return dm
}

// DB allows to get the db object for performing actions
func (dm *DatabaseManager) DB() (*sqlx.DB, error) {
	if dm.db == nil {
		return nil, ErrConnRefused
	}

	return dm.db, nil
}

func (dm *DatabaseManager) Close() error {
	if dm.db == nil {
		return ErrConnRefused
	}

	err := dm.db.Close()
	dm.db = nil
	return err
}
