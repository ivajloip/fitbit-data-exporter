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

package influxdb

import (
	"fmt"
	"sync"
	"time"

	influx "github.com/influxdata/influxdb/client/v2"
	log "github.com/sirupsen/logrus"

	"github.com/ivajloip/fitbit-data-exporter/internal/model"
	"github.com/ivajloip/fitbit-data-exporter/internal/storage"
)

const (
	tableName = "heart_reading"
	// RetryTimeout gives a default retry timeout for retryable operations
	RetryTimeout = 2 * time.Minute
	// RetryPeriod gives a default retry period for retryable operations
	RetryPeriod = 2 * time.Second
)

type influxStorage struct {
	client influx.Client

	database      string
	username      string
	tags          []string
	precision     string
	batchSize     int
	batchInterval time.Duration
	batchChan     chan *influx.Point
	flushChan     chan struct{}
	flushed       chan struct{}
	err           error
	errLock       sync.RWMutex
}

// NewStorage TODO.
func NewStorage(dataOwner, database, addr, username, password string) (storage.Storage, error) {
	c, err := influx.NewHTTPClient(influx.HTTPConfig{
		Addr:     addr,
		Username: username,
		Password: password,
	})
	if err != nil {
		return nil, err
	}
	err = RetryOnError(RetryPeriod, RetryTimeout, func() error {
		return ensureDBExists(c, database)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create InfluxDB database: %v", err.Error())
	}

	batchSize := 100
	batchInterval := 5 * time.Second
	precision := "ms"

	res := influxStorage{
		client:   c,
		username: dataOwner,

		database:      database,
		precision:     precision,
		batchSize:     batchSize,
		batchInterval: batchInterval,
		batchChan:     make(chan *influx.Point, batchSize),
		flushChan:     make(chan struct{}),
		flushed:       make(chan struct{}),
		err:           nil,
		errLock:       sync.RWMutex{},
	}

	go res.processInfluxPoints()

	return &res, nil
}

// RetryOnError runs a function every period time until it returns no error or a timeout is reached.
func RetryOnError(period, timeout time.Duration, fn func() error) error {
	startTime := time.Now()
	for {
		err := fn()
		if err == nil {
			return nil
		}
		currentTime := time.Now()
		if startTime.Add(timeout).Before(currentTime) {
			return err
		}
		log.WithError(err).WithField("period", period).Warn("failed to execute critical command, retrying...")
		time.Sleep(period)
	}
}

func (i *influxStorage) setError(e error) {
	i.errLock.Lock()
	i.err = e
	i.errLock.Unlock()
}

// Close closes the handler, taking care that all the points sent before the close are written.
// Note
func (i *influxStorage) Close() error {
	i.flushChan <- struct{}{}
	close(i.batchChan)
	<-i.flushed

	i.errLock.RLock()
	err := i.err
	i.errLock.RUnlock()

	return err
}

func (i *influxStorage) processInfluxPoints() {
	var err error
	var batch influx.BatchPoints

	ticker := time.NewTicker(i.batchInterval)
	batch, err = influx.NewBatchPoints(influx.BatchPointsConfig{
		Database:  i.database,
		Precision: i.precision,
	})
	if err != nil {
		log.Errorf("Could not create the InfluxDB batch of points: %v", err)
	}

	flushAndClear := func() {
		err := i.client.Write(batch)
		i.setError(err)

		// only clear the buffer if all data is written to the server
		if err == nil {
			batch, _ = influx.NewBatchPoints(influx.BatchPointsConfig{
				Database:  i.database,
				Precision: i.precision,
			})
		}
	}

	for {
		select {
		case <-ticker.C:
			flushAndClear()
		case p := <-i.batchChan:
			batch.AddPoint(p)
			if len(batch.Points()) >= i.batchSize {
				flushAndClear()
			}
		case <-i.flushChan:
			for p := range i.batchChan {
				batch.AddPoint(p)
			}
			flushAndClear()
			i.flushed <- struct{}{}
		}
	}
}

func ensureDBExists(client influx.Client, db string) error {
	response, err := client.Query(influx.Query{
		Command:  fmt.Sprintf("CREATE DATABASE %q", db),
		Database: db,
	})
	if err != nil {
		return err
	}

	return response.Error()
}

// SaveToDB TODO.
func (i *influxStorage) Save(data []model.HeartData) error {
	for _, d := range data {
		fields := map[string]interface{}{
			"username":   i.username,
			"bpm":        d.Value.BMP,
			"confidence": d.Value.Confidence,
		}

		pt, err := influx.NewPoint(tableName, nil, fields, d.DateTime)
		if err != nil {
			return fmt.Errorf("failed to create influx point: %v", err)
		}
		i.batchChan <- pt
	}

	return nil
}

func (i *influxStorage) IsPresent(t time.Time) (bool, error) {
	res, err := i.client.Query(influx.Query{
		Database: i.database,
		Command:  fmt.Sprintf("SELECT bpm FROM %s WHERE time > %d AND time < %d + 1d", tableName, t.UnixNano(), t.UnixNano()),
	})
	if err != nil {
		return false, err
	}
	if res.Error() != nil {
		return false, res.Error()
	}
	return len(res.Results) > 0 && len(res.Results[0].Series) > 0, nil
}
