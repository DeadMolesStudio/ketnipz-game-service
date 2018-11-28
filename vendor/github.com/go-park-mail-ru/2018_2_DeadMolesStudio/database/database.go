package database

import (
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // postgres driver

	"github.com/go-park-mail-ru/2018_2_DeadMolesStudio/logger"
)

var db *sqlx.DB

func InitDB(address, database string) *sqlx.DB {
	var err error
	db, err = sqlx.Open("postgres",
		"postgres://"+address+"/"+database+"?sslmode=disable")
	if err != nil {
		logger.Panic(err)
	}

	if err := db.Ping(); err != nil {
		logger.Panic(err)
	}

	logger.Infof("Successfully connected to %v, database %v", address, database)

	makeMigrations(db)

	return db
}

// DB allows to get the db object for performing actions
func DB() *sqlx.DB {
	return db
}
