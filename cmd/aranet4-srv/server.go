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

	"go.etcd.io/bbolt"
	"sbinet.org/x/aranet4"
)

type server struct {
	addr string // Aranet4 device address
	mux  *http.ServeMux

	mu    sync.RWMutex
	db    *bbolt.DB
	last  aranet4.Data
	plots struct {
		CO2     bytes.Buffer
		T, H, P bytes.Buffer
	}
}

func newServer(addr, dbfile string) *server {
	db, err := bbolt.Open(dbfile, 0644, &bbolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		log.Panicf("could not open aranet4 db: %+v", err)
	}

	srv := &server{
		addr: addr,
		db:   db,
		mux:  http.NewServeMux(),
	}
	srv.mux.HandleFunc("/", srv.handleRoot)
	srv.mux.HandleFunc("/update", srv.handleUpdate)
	srv.mux.HandleFunc("/plot-co2", srv.handlePlotCO2)
	srv.mux.HandleFunc("/plot-h", srv.handlePlotH)
	srv.mux.HandleFunc("/plot-p", srv.handlePlotP)
	srv.mux.HandleFunc("/plot-t", srv.handlePlotT)

	err = srv.init()
	if err != nil {
		log.Panicf("could not initialize server: %+v", err)
	}

	go srv.loop()
	return srv
}

func (srv *server) Close() error {
	return srv.db.Close()
}

func (srv *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	srv.mux.ServeHTTP(w, r)
}

func (srv *server) handleRoot(w http.ResponseWriter, r *http.Request) {
	srv.mu.RLock()
	defer srv.mu.RUnlock()

	refresh := int(srv.last.Interval.Seconds())
	if refresh == 0 {
		refresh = 10
	}
	fmt.Fprintf(w, page, refresh, srv.last.String())
}

func (srv *server) handleUpdate(w http.ResponseWriter, r *http.Request) {
	err := retry(10, func() error {
		return srv.update(-1)
	})
	if err != nil {
		fmt.Fprintf(w, "could not fetch update samples: %+v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	return
}

func (srv *server) handlePlotCO2(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "image/png")
	w.Write(srv.plots.CO2.Bytes())
}

func (srv *server) handlePlotH(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "image/png")
	w.Write(srv.plots.H.Bytes())
}

func (srv *server) handlePlotP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "image/png")
	w.Write(srv.plots.P.Bytes())
}

func (srv *server) handlePlotT(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "image/png")
	w.Write(srv.plots.T.Bytes())
}

func (srv *server) loop() {
	var (
		interval time.Duration
		err      error
	)
	err = retry(5, func() error {
		interval, err = srv.interval()
		return err
	})
	if err != nil {
		log.Panicf("could not fetch refresh frequency: %+v", err)
	}

	log.Printf("refresh frequency: %v", interval)
	tck := time.NewTicker(interval)
	defer tck.Stop()

	log.Printf("starting loop...")
	err = retry(5, func() error {
		return srv.update(-1)
	})
	if err != nil {
		log.Printf("could not update db: %+v", err)
	}
	for range tck.C {
		log.Printf("tick: %s", time.Now().UTC().Format("2006-01-02 15:04:05"))
		err := retry(5, func() error {
			return srv.update(1)
		})
		if err != nil {
			log.Printf("could not update db: %+v", err)
		}
	}
}

func retry(n int, f func() error) error {
	var err error
	for i := 0; i < n; i++ {
		err = f()
		if err != nil {
			log.Printf("retry %d/%d failed with: %+v", i+1, n, err)
			continue
		}
		return nil
	}
	return err
}
