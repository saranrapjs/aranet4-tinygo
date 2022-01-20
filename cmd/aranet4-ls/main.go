// Copyright ©2022 The aranet4 Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main // import "sbinet.org/x/aranet4/cmd/aranet4-ls"

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"sbinet.org/x/aranet4"
)

func main() {
	log.SetPrefix("aranet4: ")
	log.SetFlags(0)

	var (
		addr    = flag.String("addr", "F5:6C:BE:D5:61:47", "MAC address of Aranet4")
		verbose = flag.Bool("v", false, "enable verbose mode")

		doTimeSeries = flag.Bool("ts", false, "fetch time series")
		oname        = flag.String("o", "", "path to output file for time series")
	)

	flag.Parse()

	dev, err := aranet4.New(*addr)
	if err != nil {
		log.Fatalf("could not create aranet4 client: %+v", err)
	}
	defer dev.Close()

	if *verbose {
		name, err := dev.Name()
		if err != nil {
			log.Printf("could not get device name: %+v", err)
		}
		log.Printf("name: %q", name)

		vers, err := dev.Version()
		if err != nil {
			log.Fatalf("could not get device version: %+v", err)
		}
		log.Printf("vers: %q", vers)
	}

	data, err := dev.Read()
	if err != nil {
		log.Fatalf("could not run client: %+v", err)
	}
	fmt.Printf("%v", data)

	if *doTimeSeries {
		var (
			w     io.Writer
			flush = func() error { return nil }
		)
		switch *oname {
		case "", "-":
			w = os.Stdout
		default:
			f, err := os.Create(*oname)
			if err != nil {
				log.Fatalf("could not create output file: %+v", err)
			}
			w = f
			flush = func() error {
				return f.Close()
			}
		}

		vs, err := dev.ReadAll()
		if err != nil {
			log.Fatalf("could not read data: %+v", err)
		}
		fmt.Fprintf(w, "id;timestamp;temperature (°C);humidity (%%);pressure (hPa);CO2 (ppm)\n")
		for i, v := range vs {
			fmt.Fprintf(w, "%d;%s;%.2f;%g;%.1f;%d\n",
				i, v.Time.Format("2006-01-02 15:04:05"),
				v.T, v.H, v.P, v.CO2,
			)
		}
		err = flush()
		if err != nil {
			log.Fatalf("could not flush output file: %+v", err)
		}
	}

	err = dev.Close()
	if err != nil {
		log.Fatalf("could not close client: %+v", err)
	}
}
