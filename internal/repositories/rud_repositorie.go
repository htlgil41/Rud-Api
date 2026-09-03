package repositories

import (
	"database/sql"
	"rud-api/internal/consts"
)

type RudRepositorie struct {
	Db *sql.DB
}

func (r *RudRepositorie) InitDbMigration() error {
	_, errRowsInit := r.Db.Query(consts.INIT_DB_MIGRATION_RUD_SQLITE)
	if errRowsInit != nil {
		return errRowsInit
	}
	return nil
}

func (r *RudRepositorie) RollbackData() {
	r.Db.Exec(consts.ROLLBACK_DATABASE_RUD)
}
