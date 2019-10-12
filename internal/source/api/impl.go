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

package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/ivajloip/fitbit-data-exporter/internal/model"
	"github.com/ivajloip/fitbit-data-exporter/internal/oauth2"
	"github.com/ivajloip/fitbit-data-exporter/internal/source"
	log "github.com/sirupsen/logrus"
)

// New creates a new source.Source that is backed by a folder with json files
// representing readings for different days.
//
// The files in the folder should be named heart_rate-yyyy-mm-dd.json.
func New(client *oauth2.Client, baseURL, precision string) (source.Source, error) {
	return &reader{
		client:    client,
		baseURL:   baseURL,
		precision: precision,
	}, nil
}

type reader struct {
	client    *oauth2.Client
	baseURL   string
	precision string
}

// ReadData TODO.
func (r *reader) ReadData(t time.Time) ([]model.HeartData, error) {
	url := fmt.Sprintf("%v/%d-%0.2d-%0.2d/1d/%v/time/00:00/23:59.json", r.baseURL, t.Year(), t.Month(), t.Day(), r.precision)

	response, err := r.client.Get(url)
	if err != nil {
		log.Fatalf("get reading failed: %v", err)
	}
	b, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	var res model.HeartAPIData
	if err := json.Unmarshal(b, &res); err != nil {
		return nil, err
	}
	d := model.ToHeartData(res, t)
	log.Debugf("body: %v", d)

	return d, nil
}

func (r *reader) Close() error {
	return r.client.Close()
}
