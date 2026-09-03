package databases

import (
	"context"
	"database/sql"
	"time"
)

type SqLiteDatabase struct {
	Db *sql.DB
}

func CreateSqLiteDatabase(path_uri string) (*sql.DB, error) {
	db, errDb := sql.Open("sqlite", path_uri)
	if errDb != nil {
		return nil, errDb
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(25 * time.Minute)
	db.SetConnMaxIdleTime(time.Minute)
	return db, nil
}

func (r *SqLiteDatabase) TestConnection() bool {
	ct, cancelCt := context.WithTimeout(context.Background(), time.Second)
	defer cancelCt()

	if errPing := r.Db.PingContext(ct); errPing != nil {
		return true
	}
	return true
}
