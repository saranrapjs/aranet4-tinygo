// Copyright ©2022 The aranet4 Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"sbinet.org/x/aranet4"
)

type server struct {
	addr string // Aranet4 device address

	mu   sync.RWMutex
	data []aranet4.Data
}

func newServer(addr string) *server {
	srv := &server{addr: addr}
	go srv.loop()
	return srv
}

func (srv *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	o := new(bytes.Buffer)
	err := srv.plot(o)
	if err != nil {
		log.Printf("could not create plots: %+v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	o.WriteTo(w)
}

func (srv *server) loop() {
	interval, err := srv.interval()
	if err != nil {
		log.Panicf("could not read initial data: %+v", err)
	}

	tck := time.NewTicker(interval)
	defer tck.Stop()

	fetch := func() {
		srv.mu.Lock()
		defer srv.mu.Unlock()
		vs, err := srv.rows()
		if err != nil {
			log.Printf("could not read data: %+v", err)
			return
		}
		srv.data = vs
	}

	fetch()
	for range tck.C {
		fetch()
	}
}

func (srv *server) interval() (time.Duration, error) {
	dev, err := aranet4.New(srv.addr)
	if err != nil {
		return 0, fmt.Errorf("could not create aranet4 client: %w", err)
	}
	defer dev.Close()

	return dev.Interval()
}

func (srv *server) rows() ([]aranet4.Data, error) {
	dev, err := aranet4.New(srv.addr)
	if err != nil {
		return nil, fmt.Errorf("could not create aranet4 client: %w", err)
	}
	defer dev.Close()

	return dev.ReadAll()
}

func (srv *server) row() (aranet4.Data, error) {
	dev, err := aranet4.New(srv.addr)
	if err != nil {
		return aranet4.Data{}, fmt.Errorf("could not create aranet4 client: %w", err)
	}
	defer dev.Close()

	return dev.Read()
}
