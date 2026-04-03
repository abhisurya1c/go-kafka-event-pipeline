package infra

import (
	"database/sql"
	_ "github.com/microsoft/go-mssqldb"
	"time"
)

func NewDB(connStr string) (*sql.DB, error) {
	db, err := sql.Open("sqlserver", connStr)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(time.Hour)

	if err := db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}
