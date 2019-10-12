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

package offline

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/ivajloip/fitbit-data-exporter/internal/model"
	"github.com/ivajloip/fitbit-data-exporter/internal/source"
)

// New creates a new source.Source that is backed by a folder with json files
// representing readings for different days.
//
// The files in the folder should be named heart_rate-yyyy-mm-dd.json.
func New(dirPath string) (source.Source, error) {
	if dirPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		dirPath = homeDir + "/.local/fitbit-data-exporter/"
	}

	return &reader{dirPath}, nil
}

type reader struct {
	dirPath string
}

// ReadData TODO.
func (r *reader) ReadData(t time.Time) ([]model.HeartData, error) {
	fileName := r.dirPath + fmt.Sprintf("/heart_rate-%d-%0.2d-%0.2d.json", t.Year(), t.Month(), t.Day())
	b, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	var res []model.HeartData

	return res, json.Unmarshal(b, &res)
}

func (r *reader) Close() error {
	return nil
}
