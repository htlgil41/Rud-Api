package databases

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/denisenkom/go-mssqldb"
)

type MssqlDatabase struct {
	Db *sql.DB
}

func (r *MssqlDatabase) CreateMssqlDatabase(path_uri string) {
	db, errDb := sql.Open("sqlserver", path_uri)
	if errDb != nil {
	}

	db.SetMaxOpenConns(8)
	db.SetMaxIdleConns(8)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(time.Minute)

	r.Db = db
}

func (r *MssqlDatabase) TestConnection() bool {
	ct, cancelCt := context.WithTimeout(context.Background(), time.Second)
	defer cancelCt()

	if errPing := r.Db.PingContext(ct); errPing != nil {
		return true
	}
	return true
}
