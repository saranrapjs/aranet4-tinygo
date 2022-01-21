// Copyright ©2022 The aranet4 Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	"image/color"
	"math"

	"go-hep.org/x/hep/hplot"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
	"sbinet.org/x/aranet4"
)

func (srv *server) plot(data []aranet4.Data) error {
	var err error

	xs := make([]float64, 0, len(data))
	for _, v := range data {
		xs = append(xs, float64(v.Time.Unix()))
	}

	err = srv.plotCO2(xs, data)
	if err != nil {
		return fmt.Errorf("could not create CO2 plot: %w", err)
	}
	err = srv.plotT(xs, data)
	if err != nil {
		return fmt.Errorf("could not create T plot: %w", err)
	}
	err = srv.plotH(xs, data)
	if err != nil {
		return fmt.Errorf("could not create H plot: %w", err)
	}
	err = srv.plotP(xs, data)
	if err != nil {
		return fmt.Errorf("could not create P plot: %w", err)
	}

	return nil
}

func (srv *server) plotCO2(xs []float64, data []aranet4.Data) error {
	var (
		ys = make([]float64, 0, len(data))
	)

	for _, data := range data {
		ys = append(ys, float64(data.CO2))
	}

	c := color.NRGBA{B: 255, A: 255}
	return srv.genPlot(&srv.plots.CO2, xs, ys, "CO2 [ppm]", c)
}

func (srv *server) plotT(xs []float64, data []aranet4.Data) error {
	var (
		ys = make([]float64, 0, len(data))
	)

	for _, data := range data {
		ys = append(ys, float64(data.T))
	}

	c := color.NRGBA{R: 255, A: 255}
	return srv.genPlot(&srv.plots.T, xs, ys, "T [°C]", c)
}

func (srv *server) plotH(xs []float64, data []aranet4.Data) error {
	var (
		ys = make([]float64, 0, len(data))
	)

	for _, data := range data {
		ys = append(ys, float64(data.H))
	}

	c := color.NRGBA{G: 255, A: 255}
	return srv.genPlot(&srv.plots.H, xs, ys, "Humidity [%]", c)
}

func (srv *server) plotP(xs []float64, data []aranet4.Data) error {
	var (
		ys = make([]float64, 0, len(data))
	)

	for _, data := range data {
		ys = append(ys, float64(data.P))
	}

	c := color.NRGBA{B: 255, G: 255, A: 255}
	return srv.genPlot(&srv.plots.P, xs, ys, "Atmospheric Pressure [hPa]", c)
}

func (srv *server) genPlot(buf *bytes.Buffer, xs, ys []float64, label string, c color.NRGBA) error {

	buf.Reset()

	plt := hplot.New()
	plt.Y.Label.Text = label
	plt.X.Tick.Marker = plot.TimeTicks{Format: "2006-01-02\n15:04"}

	sca, err := hplot.NewScatter(hplot.ZipXY(xs, ys))
	if err != nil {
		return fmt.Errorf("could not create CO2 scatter plot: %w", err)
	}

	c1 := c
	c2 := c
	c2.A = 38

	sca.GlyphStyle.Color = c1
	sca.GlyphStyle.Radius = 2
	sca.GlyphStyle.Shape = draw.CircleGlyph{}

	lin, err := hplot.NewLine(hplot.ZipXY(xs, ys))
	if err != nil {
		return fmt.Errorf("could not create CO2 line plot: %w", err)
	}
	lin.LineStyle.Color = c1
	lin.FillColor = c2

	plt.Add(hplot.NewGrid(), lin, sca)

	const size = 20 * vg.Centimeter
	cnv := vgimg.PngCanvas{
		Canvas: vgimg.New(vg.Length(math.Phi)*size, size),
	}
	plt.Draw(draw.New(cnv))
	_, err = cnv.WriteTo(buf)
	if err != nil {
		return fmt.Errorf("could not create CO2 plot: %w", err)
	}

	return nil
}

const page = `
<html>
	<head>
		<title>Aranet4 monitoring</title>
		<meta http-equiv="refresh" content="%d">
	</head>

	<body>
		<pre>
%s
		</pre>
		<!-- CO2 -->
		<hr>
        <div class="row align-items-center justify-content-center">
		  <img src="/plot-co2"/>
        </div>

		<!-- Temperature -->
		<hr>
        <div class="row align-items-center justify-content-center">
		  <img src="/plot-t"/>
        </div>
		
		<!-- Humidity -->
		<hr>
        <div class="row align-items-center justify-content-center">
		  <img src="/plot-h"/>
        </div>

		<!-- Pressure -->
		<hr>
        <div class="row align-items-center justify-content-center">
		  <img src="/plot-p"/>
        </div>
	</body>
</html>
`
