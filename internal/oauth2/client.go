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

package oauth2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	log "github.com/sirupsen/logrus"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/fitbit"
)

var (
	// ErrMissingClientInformation represents missing configuration information
	// that is needed to create a new client.
	ErrMissingClientInformation = errors.New("missing clientID or clientSecret")
)

// Config contains the data needed to store state for invoking fitbit API.
type Config struct {
	Token        *oauth2.Token
	ClientID     string
	ClientSecret string
	Scopes       []string
}

// Client TODO.
type Client struct {
	config     Config
	configFile string
	client     *http.Client
	source     oauth2.TokenSource
}

// Close prepares the client to be deleted.
func (c *Client) Close() error {
	var err error
	c.config.Token, err = c.source.Token()
	if err != nil {
		return err
	}
	err = c.config.WriteToFile(c.configFile)
	return err
}

// Get proxies the get request to the target adding oauth2 authenticatoin.
func (c *Client) Get(url string) (*http.Response, error) {
	return c.client.Get(url)
}

// New returns an Client whose Get method is configured to work with fitbit
// oauth2 authentication.
func New(configFile, bindAddr string, c Config) (*Client, error) {
	ctx := context.Background()
	if _, err := os.Stat(configFile); err == nil {
		c, err = readConfFromFile(configFile, c)
		if err != nil {
			return nil, err
		}
	}
	conf := &oauth2.Config{
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		Scopes:       c.Scopes,
		Endpoint:     fitbit.Endpoint,
	}
	var err error
	source := conf.TokenSource(ctx, c.Token)
	if c.Token, err = source.Token(); err != nil {
		if c.Token, err = authCode(bindAddr, conf); err != nil {
			return nil, err
		}
		source = conf.TokenSource(ctx, c.Token)
	}
	_ = c.WriteToFile(configFile)
	res := Client{
		configFile: configFile,
		config:     c,
		client:     conf.Client(ctx, c.Token),
		source:     source,
	}
	return &res, nil
}

func readConfFromFile(configFile string, curConfig Config) (Config, error) {
	var config Config
	buf, err := ioutil.ReadFile(configFile)
	if err != nil {
		return config, err
	}
	err = json.Unmarshal(buf, &config)
	if err != nil {
		return config, err
	}
	if curConfig.ClientID != "" && curConfig.ClientSecret != "" && len(curConfig.Scopes) > 0 {
		curConfig.Token = config.Token
		return curConfig, nil
	}

	return config, nil
}

func authCode(bindAddr string, conf *oauth2.Config) (*oauth2.Token, error) {
	codeCh := make(chan string)
	http.HandleFunc("/auth/fitbit/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		codeCh <- code
		_, _ = w.Write([]byte(code))
	})
	srv := http.Server{
		Addr: bindAddr,
	}
	go func() {
		_ = srv.ListenAndServe()
	}()
	if conf.ClientID == "" || conf.ClientSecret == "" {
		return nil, ErrMissingClientInformation
	}
	url := conf.AuthCodeURL("state", oauth2.AccessTypeOffline)
	fmt.Printf("Visit the URL for the auth dialog: %v\n", url)
	code := <-codeCh
	ctx := context.Background()
	_ = srv.Shutdown(ctx)
	ctx = context.Background()

	return conf.Exchange(ctx, code)
}

// WriteToFile persists the config to a file.
func (c *Config) WriteToFile(configFile string) error {
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	log.WithField("conf", string(b)).Debug("Conf file")
	log.WithField("conf-file", configFile).Debug("Writing conf file")

	return ioutil.WriteFile(configFile, b, 0644)
}
