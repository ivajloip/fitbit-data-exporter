// Copyright 2019 Ivaylo Petrov. All rights reserved.
//
// This file is part of Fitbit Data Exporter.
//
// Fitbit Data Exporter is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Fitbit Data Exporter is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Fitbit Data Exporter.  If not, see <https://www.gnu.org/licenses/>.

package postgresql

import (
	"database/sql"
	"fmt"

	// register postgresql driver
	_ "github.com/lib/pq"
	migrate "github.com/rubenv/sql-migrate"

	"github.com/gocraft/dbr/v2"
	log "github.com/sirupsen/logrus"
)

const (
	postgresMaxOpenConns = 10
)

// OpenSQLDB opens a database connection pool to a postgresql database and
// verifies that it functions correctly.
func OpenSQLDB(dsn string) (*dbr.Session, error) {
	var db *dbr.Session
	conn, err := dbr.Open("postgres", dsn, nil)
	if err != nil {
		return db, err
	}
	conn.SetMaxOpenConns(postgresMaxOpenConns)
	if err := conn.Ping(); err != nil {
		return db, fmt.Errorf("ping database error, will retry in 2s: %s", err)
	}
	db = conn.NewSession(nil)

	return db, migrationsUp(db.DB)
}

func migrationsUp(db *sql.DB) error {
	migrations := &migrate.MemoryMigrationSource{
		Migrations: []*migrate.Migration{
			&migrate.Migration{
				Id: "123",
				Up: []string{`CREATE TABLE heart_reading (
  id bigserial primary key,
  username varchar(256) not null,
	time timestamp with time zone not null,
	value integer not null,
	confidence integer not null
				)`},
				Down: []string{"DROP TABLE heart_reading"},
			},
		},
	}

	n, err := migrate.Exec(db, "postgres", migrations, migrate.Up)
	log.WithField("nb", n).Debug("Migrations run")

	return err
}
