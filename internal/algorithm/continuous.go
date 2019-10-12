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
	"sync"
	"time"

	"github.com/ivajloip/fitbit-data-exporter/internal/source"
	"github.com/ivajloip/fitbit-data-exporter/internal/storage"
)

type continuous struct {
	cancel func()
	ctx    context.Context
	wg     sync.WaitGroup
	ticker *time.Ticker

	currAlg *DefaultAlg
}

// NewContinuous TODO.
func NewContinuous(since time.Time, source source.Source, storage storage.Storage) Alg {
	ctx, cancel := context.WithCancel(context.Background())
	return &continuous{
		cancel:  cancel,
		ctx:     ctx,
		currAlg: New(since, source, storage),
	}
}

// Run TODO.
func (d *continuous) Run() error {
	d.wg.Add(1)
	defer d.wg.Done()
	d.ticker = time.NewTicker(24 * time.Hour)
	since := d.currAlg.since
	for {
		d.currAlg.since = since

		if err := d.currAlg.Run(); err != nil {
			return err
		}
		select {
		case <-d.ctx.Done():
			return nil
		case <-d.ticker.C:
			n := time.Now()
			since = time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, n.Location())
		}
	}
}

// Close TODO.
func (d *continuous) Close() error {
	d.ticker.Stop()
	d.cancel()
	_ = d.currAlg.Close()
	d.wg.Wait()

	return nil
}
