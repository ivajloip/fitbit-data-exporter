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

package model

import (
	"time"
)

// DateTime TODO.
type DateTime time.Time

// UnmarshalJSON TODO.
func (d *DateTime) UnmarshalJSON(b []byte) error {
	s := string(b)
	// remove quotes
	if len(s) > 2 {
		s = s[1 : len(s)-1]
	}
	r, err := time.Parse("15:04:05", s)
	*d = DateTime(r)

	return err
}

// MarshalJSON TODO.
func (d DateTime) MarshalJSON() ([]byte, error) {
	r := time.Time(d).Format("01/02/06 15:04:05")

	return []byte(r), nil
}

// HeartData TODO.
type HeartData struct {
	DateTime time.Time `json:"dateTime"`
	Value    Value     `json:"value"`
}

// Value TODO.
type Value struct {
	BMP        int `json:"bpm"`
	Confidence int `json:"confidence"`
}

// HeartAPIData TODO.
type HeartAPIData struct {
	ActivitiesHeart          []ActivitiesHeart        `json:"activities-heart"`
	ActivitiesHeartInteraday ActivitiesHeartInteraday `json:"activities-heart-intraday"`
}

// ActivitiesHeart TODO.
type ActivitiesHeart struct {
	Date  string      `json:"dateTime"`
	Zones []HeartZone `json:"heartRateZones"`
	Value string      `json:"value"`
}

// HeartZone TODO.
type HeartZone struct {
	Cal     float32 `json:"caloriesOut"`
	Max     uint8   `json:"max"`
	Min     uint8   `json:"min"`
	Minutes uint16  `json:"minutes"`
	Name    string  `json:"name"`
}

// ActivitiesHeartInteraday TODO.
type ActivitiesHeartInteraday struct {
	Dataset []APIValue `json:"dataset"`
}

// APIValue TODO.
type APIValue struct {
	Time  DateTime `json:"time"`
	Value int      `json:"value"`
}

// ToHeartData TODO.
func ToHeartData(d HeartAPIData, t time.Time) []HeartData {
	var res []HeartData
	for _, r := range d.ActivitiesHeartInteraday.Dataset {
		rt := time.Time(r.Time)
		ts := time.Date(t.Year(), t.Month(), t.Day(), rt.Hour(), rt.Minute(), rt.Second(), 0, t.Location())
		res = append(res, HeartData{
			DateTime: time.Time(ts),
			Value: Value{
				BMP:        r.Value,
				Confidence: 1,
			},
		})
	}

	return res
}
