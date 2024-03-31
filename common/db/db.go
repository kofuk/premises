package db

import (
	"database/sql"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

func NewClient(addr, user, password, database string) *bun.DB {
	conn := pgdriver.NewConnector(
		pgdriver.WithAddr(addr),
		pgdriver.WithUser(user),
		pgdriver.WithPassword(password),
		pgdriver.WithDatabase(database),
		pgdriver.WithInsecure(true),
		pgdriver.WithConnParams(map[string]interface{}{
			"TimeZone": "Etc/UTC",
		}),
	)
	return bun.NewDB(sql.OpenDB(conn), pgdialect.New())
}
