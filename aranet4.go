// Copyright ©2022 The aranet4 Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package aranet4 // import "sbinet.org/x/aranet4"

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	timeFmt = "2006-01-02 15:04:05 UTC"
)

const (
	uuidDeviceService          = "f0cd1400-95da-4f4b-9ac8-aa55d312af0c"
	uuidWriteCmd               = "f0cd1402-95da-4f4b-9ac8-aa55d312af0c"
	uuidReadSample             = "f0cd1503-95da-4f4b-9ac8-aa55d312af0c"
	uuidReadAll                = "f0cd3001-95da-4f4b-9ac8-aa55d312af0c"
	uuidReadInterval           = "f0cd2002-95da-4f4b-9ac8-aa55d312af0c"
	uuidReadTimeSeries         = "f0cd2003-95da-4f4b-9ac8-aa55d312af0c"
	uuidReadSecondsSinceUpdate = "f0cd2004-95da-4f4b-9ac8-aa55d312af0c"
	uuidReadTotalReadings      = "f0cd2001-95da-4f4b-9ac8-aa55d312af0c"

	uuidGenericService = "00001800-0000-1000-8000-00805f9b34fb"

	uuidCommonService              = "0000180a-0000-1000-8000-00805f9b34fb"
	uuidCommonReadManufacturerName = "00002a29-0000-1000-8000-00805f9b34fb"
	uuidCommonReadModelNumber      = "00002a24-0000-1000-8000-00805f9b34fb"
	uuidCommonReadSerialNumber     = "00002a25-0000-1000-8000-00805f9b34fb"
	uuidCommonReadHWRevision       = "00002a27-0000-1000-8000-00805f9b34fb"
	uuidCommonReadSWRevision       = "00002a28-0000-1000-8000-00805f9b34fb"
	uuidCommonReadBattery          = "00002a19-0000-1000-8000-00805f9b34fb"
)

const (
	paramT   = 1
	paramH   = 2
	paramP   = 3
	paramCO2 = 4
)

var (
	// ErrNoData indicates a missing data point.
	// This may happen during sensor calibration.
	ErrNoData = errors.New("aranet4: no data")
)

// Quality gives a general assessment of air quality (green/yellow/red).
//  - green:  [   0 - 1000) ppm
//  - yellow: [1000 - 1400) ppm
//  - red:    [1400 -  ...) ppm
type Quality int

func (st Quality) String() string {
	switch st {
	case 1:
		return "green"
	case 2:
		return "yellow"
	case 3:
		return "red"
	default:
		return fmt.Sprintf("Quality(%d)", int(st))
	}
}

func qualityFrom(co2 int) Quality {
	switch {
	case co2 < 1000:
		return 1
	case co2 < 1400:
		return 2
	default:
		return 3
	}
}

// Data holds measured data samples provided by Aranet4.
type Data struct {
	H, P, T float64
	CO2     int
	Battery int
	Quality Quality

	Interval time.Duration
	Time     time.Time
}

func (data Data) String() string {
	var o strings.Builder
	fmt.Fprintf(&o, "CO2:         %d ppm\n", data.CO2)
	fmt.Fprintf(&o, "temperature: %g°C\n", data.T)
	fmt.Fprintf(&o, "pressure:    %g hPa\n", data.P)
	fmt.Fprintf(&o, "humidity:    %g%%\n", data.H)
	fmt.Fprintf(&o, "quality:     %v\n", data.Quality)
	fmt.Fprintf(&o, "battery:     %d%%\n", data.Battery)
	fmt.Fprintf(&o, "interval:    %v\n", data.Interval)
	fmt.Fprintf(&o, "time-stamp:  %v\n", data.Time.UTC().Format(timeFmt))
	return o.String()
}
