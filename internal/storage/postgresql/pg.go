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
	"time"

	dbr "github.com/gocraft/dbr/v2"
	log "github.com/sirupsen/logrus"

	"github.com/ivajloip/fitbit-data-exporter/internal/model"
	"github.com/ivajloip/fitbit-data-exporter/internal/storage"
)

const tableName = "heart_reading"

type pgStorage struct {
	s        *dbr.Session
	username string
}

// NewStorage TODO.
func NewStorage(username string, s *dbr.Session) storage.Storage {
	return &pgStorage{
		username: username,
		s:        s,
	}
}

// SaveToDB TODO.
func (p *pgStorage) Save(data []model.HeartData) error {
	for _, d := range data {
		ts := d.DateTime
		_, err := p.s.InsertInto(tableName).
			Columns("username", "time", "value", "confidence").
			Values(p.username, ts, d.Value.BMP, d.Value.Confidence).
			Exec()
		if err != nil {
			log.WithError(err).Infof("failed to add: %v %v %v", p.username, ts.Add(3*time.Hour), d.Value)
		}
	}

	return nil
}

func (p *pgStorage) IsPresent(t time.Time) (bool, error) {
	var res int
	err := p.s.Select("count(*)").From(tableName).Where("time - ? > interval '0 minutes' AND time - ? < interval '23 hours'", t, t).LoadOne(&res)

	return res > 0, err
}

func (p *pgStorage) Close() error {
	return nil
}
