// Copyright ©2022 The aranet4 Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main // import "sbinet.org/x/aranet4/cmd/aranet4-srv"

import (
	"flag"
	"log"
	"net/http"
)

func main() {
	log.SetPrefix("aranet4: ")
	log.SetFlags(0)

	var (
		addr  = flag.String("addr", ":8080", "[host]:addr to serve")
		devID = flag.String("device", "F5:6C:BE:D5:61:47", "MAC address of Aranet4")
		db    = flag.String("db", "data.db", "path to DB file")
	)

	flag.Parse()

	xmain(*addr, *devID, *db)
}

func xmain(addr, devID, db string) {
	srv := newServer(devID, db)
	defer srv.Close()

	err := http.ListenAndServe(addr, srv)
	if err != nil {
		log.Panicf("could not serve %q: %+v", addr, err)
	}
}
