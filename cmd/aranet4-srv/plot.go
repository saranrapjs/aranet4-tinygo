// Copyright ©2022 The aranet4 Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image/color"
	"io"
	"math"
	"strings"

	"go-hep.org/x/hep/hplot"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
)

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

	data, err := srv.row()
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
