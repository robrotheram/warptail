package utils

import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/mysqldialect"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/driver/sqliteshim"
)

type DBType string

const SQLITE = DBType("sqlite")
const POSTGRES = DBType("postgres")
const MYSQL = DBType("mysql")

type DatabaseConfig struct {
	ConnectionType   DBType `yaml:"connection_type"`
	ConnectionString string `yaml:"connection"`
}

func NewDatabase(config DatabaseConfig) (*bun.DB, error) {
	switch config.ConnectionType {
	case SQLITE:
		sqldb, err := sql.Open(sqliteshim.ShimName, config.ConnectionString)
		if err != nil {
			return nil, err
		}
		return bun.NewDB(sqldb, sqlitedialect.New()), nil
	case POSTGRES:
		sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(config.ConnectionString)))
		return bun.NewDB(sqldb, pgdialect.New()), nil
	case MYSQL:
		sqldb, err := sql.Open("mysql", config.ConnectionString)
		if err != nil {
			return nil, err
		}
		return bun.NewDB(sqldb, mysqldialect.New()), nil
	default:
		sqldb, err := sql.Open(sqliteshim.ShimName, "file::memory:?cache=shared")
		if err != nil {
			return nil, err
		}
		return bun.NewDB(sqldb, sqlitedialect.New()), nil
	}
}
