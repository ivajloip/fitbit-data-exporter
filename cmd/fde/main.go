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

package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	"github.com/ivajloip/fitbit-data-exporter/internal/algorithm"
	client "github.com/ivajloip/fitbit-data-exporter/internal/oauth2"
	"github.com/ivajloip/fitbit-data-exporter/internal/source"
	"github.com/ivajloip/fitbit-data-exporter/internal/source/api"
	"github.com/ivajloip/fitbit-data-exporter/internal/source/offline"
	"github.com/ivajloip/fitbit-data-exporter/internal/storage"
	"github.com/ivajloip/fitbit-data-exporter/internal/storage/influxdb"
	"github.com/ivajloip/fitbit-data-exporter/internal/storage/postgresql"
)

var version string // set by the compiler

func main() {
	confDir, err := os.UserConfigDir()
	assertNoError(err, "failed to get user config dir")

	app := cli.NewApp()
	app.Name = "FitbitDataExporter"
	app.Usage = "Fitbit Data Exporter"
	app.Version = fmt.Sprintf("%s", version)
	app.Copyright = "See https://github.com/ivajloip/fitbit-data-exporter for copyright information"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "username, u",
			Usage:  "Username to be associated with the data",
			EnvVar: "FDE_USERNAME",
		},
		cli.StringFlag{
			Name:   "starting-date",
			Usage:  "Starting date (ex 2019/06/01 or 48h, which is 48h before the current time)",
			EnvVar: "FDE_START_DATE",
		},
		cli.StringFlag{
			Name:   "postgresql-dsn, psql",
			Usage:  "DSN for the connection to postgresql",
			EnvVar: "FDE_POSTGRESQL_DSN",
		},
		cli.StringFlag{
			Name:   "influxdb-url",
			Usage:  "InfluxDB URL",
			EnvVar: "FDE_INFLUXDB_URL",
		},
		cli.StringFlag{
			Name:   "influxdb-database",
			Value:  "fitbit-data",
			Usage:  "InfluxDB database name",
			EnvVar: "FDE_INFLUXDB_DATABASE",
		},
		cli.StringFlag{
			Name:   "influxdb-username",
			Usage:  "InfluxDB Username",
			EnvVar: "FDE_INFLUXDB_USERNAME",
		},
		cli.StringFlag{
			Name:   "influxdb-password",
			Usage:  "InfluxDB Password",
			EnvVar: "FDE_INFLUXDB_PASSWORD",
		},
		cli.IntFlag{
			Name:   "log-level",
			Value:  4,
			Usage:  "debug=5, info=4, warning=3, error=2, fatal=1, panic=0",
			EnvVar: "FDE_LOG_LEVEL",
		},
		cli.StringFlag{
			Name:   "record-cpu-statistics-path",
			Usage:  "record CPU statistics path (ex. /tmp/fde_cpu.prof)",
			EnvVar: "FDE_RECORD_CPU_STATISTICS_PATH",
		},
	}
	app.Commands = []cli.Command{
		cli.Command{
			Name:    "offline",
			Aliases: []string{"off"},
			Usage:   "Reads data from unpackaged download of all personal data and uploads it in the provided database",
			Action:  runOffline,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:   "dirpath",
					Usage:  "Path to folder containing the json heartrate data files",
					EnvVar: "FDE_DIR_PATH",
				},
			},
		},
		cli.Command{
			Name:    "api",
			Aliases: []string{"a"},
			Usage:   "Reads data from online api and uploads it in the provided database",
			Action:  runAPI,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:   "conf-file",
					Value:  confDir + "/fitbit-oauth2.json",
					Usage:  "",
					EnvVar: "FDE_API_OAUTH2_CONF_FILE",
				},
				cli.StringFlag{
					Name:   "bind-addr",
					Value:  "127.0.0.1:5556",
					Usage:  "",
					EnvVar: "FDE_API_BIND_ADDR",
				},
				cli.StringFlag{
					Name:   "client-id",
					Usage:  "",
					EnvVar: "FDE_API_CLIENT_ID",
				},
				cli.StringFlag{
					Name:   "client-secret",
					Usage:  "",
					EnvVar: "FDE_API_CLIENT_SECRET",
				},
				cli.StringFlag{
					Name:   "base-url",
					Value:  "https://api.fitbit.com/1/user/-/activities/heart/date",
					Usage:  "",
					EnvVar: "FDE_API_BASE_URL",
				},
				cli.StringFlag{
					Name:   "precision",
					Value:  "1sec",
					Usage:  "",
					EnvVar: "FDE_API_PRECISION",
				},
				cli.BoolFlag{
					Name:   "daemon",
					Usage:  "",
					EnvVar: "FDE_API_DAEMON",
				},
			},
		},
	}
	err = app.Run(os.Args)
	if err != nil {
		log.WithError(err).Fatal("Execution error")
	}
}

func assertNoError(err error, template string, params ...interface{}) {
	if err != nil {
		log.WithError(err).Fatalf(template, params...)
	}
}

func runAPI(c *cli.Context) error {
	log.SetLevel(log.Level(c.GlobalInt("log-level")))
	storage := mustCreateStorage(c)
	source, err := getOAuth2Source(c)
	if err != nil {
		return err
	}

	since := mustGetStartingDate(c)
	assertNoError(err, "failed to open source")
	var alg algorithm.Alg
	if c.Bool("daemon") {
		alg = algorithm.NewContinuous(since, source, storage)
	} else {
		alg = algorithm.New(since, source, storage)
	}
	defer func() {
		_ = alg.Close()
	}()

	return runWithSignalHandling(alg, c)
}

func getOAuth2Source(c *cli.Context) (source.Source, error) {
	confFile := c.String("conf-file")
	bindAddr := c.String("bind-addr")
	conf := client.Config{
		ClientID:     c.String("client-id"),
		ClientSecret: c.String("client-secret"),
		Scopes:       []string{"heartrate"},
	}
	cl, err := client.New(confFile, bindAddr, conf)
	if err == client.ErrMissingClientInformation {
		return nil, fmt.Errorf("invalid credentials configuration: %v", err)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to build a client: %v", err)
	}
	baseURL := c.String("base-url")
	precision := c.String("precision")

	return api.New(cl, baseURL, precision)
}

func mustGetStartingDate(c *cli.Context) time.Time {
	startingDate := c.GlobalString("starting-date")
	since, err := time.Parse("2006/01/02", startingDate)
	if err != nil {
		d, err := time.ParseDuration(startingDate)
		assertNoError(err, "failed to parse starting-date")
		since = time.Now().Add(-d)
		since = time.Date(since.Year(), since.Month(), since.Day(), 0, 0, 0, 0, since.Location())
	}

	return since
}

func mustCreateStorage(c *cli.Context) storage.Storage {
	owner := c.GlobalString("username")
	dsn := c.GlobalString("postgresql-dsn")
	if len(dsn) > 0 {
		db, err := postgresql.OpenSQLDB(dsn)
		assertNoError(err, "failed to open pg db")
		return postgresql.NewStorage(owner, db)
	}

	addr := c.GlobalString("influxdb-url")
	db := c.GlobalString("influxdb-database")
	user := c.GlobalString("influxdb-username")
	pass := c.GlobalString("influxdb-password")
	s, err := influxdb.NewStorage(owner, db, addr, user, pass)
	assertNoError(err, "failed to open influxdb db")

	return s
}

func runWithSignalHandling(runner algorithm.Alg, c *cli.Context) error {
	endCh := make(chan error)
	go func() {
		endCh <- runner.Run()
	}()
	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGUSR2)

	for {
		select {
		case sig := <-sigChan:
			log.WithField("sig", sig).Warn("received a signal")
			switch sig {
			case syscall.SIGUSR2:
				if log.GetLevel() == log.DebugLevel {
					log.SetLevel(log.Level(c.Int("log-level")))
				} else {
					log.SetLevel(log.DebugLevel)
				}
			default:
				log.Warn("stopping gracefully...")
				return fmt.Errorf("received signal: %v", sig.String())
			}
		case err := <-endCh:
			return err
		}
	}
}

func runOffline(c *cli.Context) error {
	log.SetLevel(log.Level(c.GlobalInt("log-level")))
	since := mustGetStartingDate(c)
	dirPath := c.String("filepath")
	source, err := offline.New(dirPath)
	assertNoError(err, "failed to open source")
	storage := mustCreateStorage(c)

	alg := algorithm.New(since, source, storage)

	return runWithSignalHandling(alg, c)
}
