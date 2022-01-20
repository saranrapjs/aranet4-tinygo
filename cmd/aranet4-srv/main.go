// Copyright ©2022 The aranet4 Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main // import "sbinet.org/x/aranet4/cmd/aranet4-srv"

import (
	"bytes"
	"encoding/base64"
	"flag"
	"fmt"
	"image/color"
	"io"
	"log"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"go-hep.org/x/hep/hplot"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
	"sbinet.org/x/aranet4"
)

func main() {
	log.SetPrefix("aranet4: ")
	log.SetFlags(0)

	var (
		addr  = flag.String("addr", ":8080", "[host]:addr to serve")
		devID = flag.String("device", "F5:6C:BE:D5:61:47", "MAC address of Aranet4")
	)

	flag.Parse()

	srv := newServer(*devID)
	err := http.ListenAndServe(*addr, srv)
	if err != nil {
		log.Fatalf("could not serve %q: %+v", *addr, err)
	}
}

type server struct {
	dev *aranet4.Device

	mu   sync.RWMutex
	data []aranet4.Data
}

func newServer(addr string) *server {
	dev, err := aranet4.New(addr)
	if err != nil {
		log.Panicf("could not create aranet4 client: %+v", err)
	}
	srv := &server{dev: dev}
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
	data, err := srv.dev.Read()
	if err != nil {
		log.Panicf("could not read initial data: %+v", err)
	}

	tck := time.NewTicker(data.Interval)
	defer tck.Stop()

	fetch := func() {
		srv.mu.Lock()
		defer srv.mu.Unlock()
		vs, err := srv.dev.ReadAll()
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

func (srv *server) plot(w io.Writer) error {
	srv.mu.RLock()
	defer srv.mu.RUnlock()

	xs := make([]float64, 0, len(srv.data))
	for _, data := range srv.data {
		xs = append(xs, float64(data.Time.Unix()))
	}

	plotCO2, err := srv.plotCO2(xs)
	if err != nil {
		return fmt.Errorf("could not create CO2 plot: %w", err)
	}
	plotT, err := srv.plotT(xs)
	if err != nil {
		return fmt.Errorf("could not create T plot: %w", err)
	}
	plotH, err := srv.plotH(xs)
	if err != nil {
		return fmt.Errorf("could not create H plot: %w", err)
	}
	plotP, err := srv.plotP(xs)
	if err != nil {
		return fmt.Errorf("could not create P plot: %w", err)
	}

	data, err := srv.dev.Read()
	if err != nil {
		return fmt.Errorf("could not read last sample: %w", err)
	}
	var last strings.Builder
	fmt.Fprintf(&last, "%v", data)

	fmt.Fprintf(w, page, last.String(), plotCO2, plotT, plotH, plotP)
	return nil
}

func (srv *server) plotCO2(xs []float64) (string, error) {
	var (
		ys = make([]float64, 0, len(srv.data))
	)

	for _, data := range srv.data {
		ys = append(ys, float64(data.CO2))
	}

	c := color.NRGBA{B: 255, A: 255}
	return srv.genPlot(xs, ys, "CO2 [ppm]", c)
}

func (srv *server) plotT(xs []float64) (string, error) {
	var (
		ys = make([]float64, 0, len(srv.data))
	)

	for _, data := range srv.data {
		ys = append(ys, float64(data.T))
	}

	c := color.NRGBA{R: 255, A: 255}
	return srv.genPlot(xs, ys, "T [°C]", c)
}

func (srv *server) plotH(xs []float64) (string, error) {
	var (
		ys = make([]float64, 0, len(srv.data))
	)

	for _, data := range srv.data {
		ys = append(ys, float64(data.H))
	}

	c := color.NRGBA{G: 255, A: 255}
	return srv.genPlot(xs, ys, "Humidity [%]", c)
}

func (srv *server) plotP(xs []float64) (string, error) {
	var (
		ys = make([]float64, 0, len(srv.data))
	)

	for _, data := range srv.data {
		ys = append(ys, float64(data.P))
	}

	c := color.NRGBA{B: 255, G: 255, A: 255}
	return srv.genPlot(xs, ys, "Atmospheric Pressure [hPa]", c)
}

func (srv *server) genPlot(xs, ys []float64, label string, c color.NRGBA) (string, error) {

	plt := hplot.New()
	plt.Y.Label.Text = label
	plt.X.Tick.Marker = plot.TimeTicks{Format: "2006-01-02\n15:04"}

	sca, err := hplot.NewScatter(hplot.ZipXY(xs, ys))
	if err != nil {
		return "", fmt.Errorf("could not create CO2 scatter plot: %w", err)
	}

	c1 := c
	c2 := c
	c2.A = 38

	sca.GlyphStyle.Color = c1
	sca.GlyphStyle.Radius = 2
	sca.GlyphStyle.Shape = draw.CircleGlyph{}

	lin, err := hplot.NewLine(hplot.ZipXY(xs, ys))
	if err != nil {
		return "", fmt.Errorf("could not create CO2 line plot: %w", err)
	}
	lin.LineStyle.Color = c1
	lin.FillColor = c2

	plt.Add(hplot.NewGrid(), lin, sca)

	const size = 20 * vg.Centimeter
	buf := new(bytes.Buffer)
	cnv := vgimg.PngCanvas{
		Canvas: vgimg.New(vg.Length(math.Phi)*size, size),
	}
	plt.Draw(draw.New(cnv))
	_, err = cnv.WriteTo(buf)
	if err != nil {
		return "", fmt.Errorf("could not create CO2 plot: %w", err)
	}

	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

const page = `
<html>
	<head>
		<title>Aranet4 monitoring</title>
	</head>

	<body>
		<pre>
%s
		</pre>
		<!-- CO2 -->
		<hr>
        <div class="row align-items-center justify-content-center">
		  <img src="data:image/png;base64,%s"/>
        </div>

		<!-- Temperature -->
		<hr>
        <div class="row align-items-center justify-content-center">
		  <img src="data:image/png;base64,%s"/>
        </div>
		
		<!-- Humidity -->
		<hr>
        <div class="row align-items-center justify-content-center">
		  <img src="data:image/png;base64,%s"/>
        </div>

		<!-- Pressure -->
		<hr>
        <div class="row align-items-center justify-content-center">
		  <img src="data:image/png;base64,%s"/>
        </div>
	</body>
</html>
`
