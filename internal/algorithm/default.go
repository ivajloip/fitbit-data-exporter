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

package algorithm

import (
	"context"
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/ivajloip/fitbit-data-exporter/internal/source"
	"github.com/ivajloip/fitbit-data-exporter/internal/storage"
)

// Alg TODO.
type Alg interface {
	Run() error
	Close() error
}

// DefaultAlg TODO.
type DefaultAlg struct {
	cancel  func()
	ctx     context.Context
	wg      sync.WaitGroup
	since   time.Time
	source  source.Source
	storage storage.Storage
}

// New TODO.
func New(since time.Time, source source.Source, storage storage.Storage) *DefaultAlg {
	ctx, cancel := context.WithCancel(context.Background())
	return &DefaultAlg{
		cancel:  cancel,
		ctx:     ctx,
		since:   since,
		source:  source,
		storage: storage,
	}
}

// Run TODO.
func (d *DefaultAlg) Run() error {
	d.wg.Add(1)
	defer d.wg.Done()
	now := time.Now()
	for i := 0; ; i++ {
		select {
		case <-d.ctx.Done():
			return d.ctx.Err()
		default:
		}
		t := d.since.Add(time.Duration(i) * 24 * time.Hour)
		if !t.Before(now.Add(-24 * time.Hour)) {
			break
		}
		log.WithField("ts", t).Info("reading data for date")
		if present, err := d.storage.IsPresent(t); err != nil {
			return fmt.Errorf("failed to verify presence: %v", err)
		} else if present {
			log.WithField("ts", t).Debug("date already present, skipping...")
			continue
		}
		data, err := d.source.ReadData(t)
		if err != nil {
			return fmt.Errorf("failed to read data: %v", err)
		}
		log.WithField("ts", t).Debug("data successfully read")
		if err := d.storage.Save(data); err != nil {
			return fmt.Errorf("failed to save data: %v", err)
		}
	}

	return nil
}

// Close TODO.
func (d *DefaultAlg) Close() error {
	d.cancel()
	_ = d.source.Close()
	_ = d.storage.Close()
	d.wg.Wait()

	return nil
}
