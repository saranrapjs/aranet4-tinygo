// Copyright ©2022 The aranet4 Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package aranet4

import (
	"bytes"
	"encoding/binary"
	// "errors"
	"fmt"
	// "io"
	"time"

	// "github.com/muka/go-bluetooth/bluez/profile/adapter"
	// "github.com/muka/go-bluetooth/bluez/profile/device"
	"tinygo.org/x/bluetooth"
)

type Device struct {
	addr string
	dev  *bluetooth.Device
	scan bluetooth.ScanResult
	svc *bluetooth.DeviceService
}

func New(addr string) (*Device, error) {
	ad := bluetooth.DefaultAdapter
	err := ad.Enable()
	if err != nil {
		return nil, fmt.Errorf("could not set default adapter power: %w", err)
	}

	var foundDevice bluetooth.ScanResult
	err = ad.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
		if (result.Address.String() != addr) {
			return
		}
		foundDevice = result			
		// Stop the scan.
		adapter.StopScan()
	})
	if err != nil {
		return nil, fmt.Errorf("could not start a scan %q: %w", addr, err)
	}

	dev, err := ad.Connect(foundDevice.Address, bluetooth.ConnectionParams{})
	if err != nil {
		return nil, fmt.Errorf("could not connect to %q: %w", addr, err)
	}

	return &Device{addr: addr, dev: dev}, nil
}

func (dev *Device) Close() error {
	if dev.dev == nil {
		return nil
	}
	err := dev.dev.Disconnect()
	if err != nil {
		return fmt.Errorf("could not disconnect: %w", err)
	}
	return nil
}

func (dev *Device) Name() (string, error) {
	return dev.scan.LocalName(), nil
}

func toUUID(str string) bluetooth.UUID {
	u, _ := bluetooth.ParseUUID(str)
	return u
}

func (dev *Device) getCharByUUID(uuidstr string) (*bluetooth.DeviceCharacteristic, error) {
	uuid := toUUID(uuidstr)
	if dev.svc == nil {
		svcs, err := dev.dev.DiscoverServices(nil)		
		if err != nil {
			return nil, fmt.Errorf("could not find service: %w", err)
		}
		for _, s := range svcs {
			if s.UUID() == toUUID(uuidDeviceService) {
				dev.svc = &s
			}
		}
	}
	if dev.svc == nil {
		return nil, fmt.Errorf("no service found", nil)
	}
	chars, err := dev.svc.DiscoverCharacteristics([]bluetooth.UUID{uuid})
	if err != nil {
		return nil, fmt.Errorf("could not get characteristic %q: %w", uuidCommonReadSWRevision, err)
	}
	var foundChar bluetooth.DeviceCharacteristic
	for _, char := range chars {
		if char.UUID() == uuid {
			foundChar = char
		}
	}
	return &foundChar, nil
}

func (dev *Device) Version() (string, error) {
	char, err := dev.getCharByUUID(uuidCommonReadSWRevision)
	if err != nil {
		return "", fmt.Errorf("could not get characteristic %q: %w", uuidCommonReadSWRevision, err)
	}
	raw := make([]byte, 255)
	if n, err := char.Read(raw); err != nil || n == 0 {
		return "", fmt.Errorf("could not read device name: %w", err)
	}
	return string(raw), nil
}

func (dev *Device) Read() (Data, error) {
	var data Data
	c, err := dev.getCharByUUID(uuidReadAll)
	if err != nil {
		return data, fmt.Errorf("could not get characteristic %q: %w", uuidReadAll, err)
	}

	raw := make([]byte, 255)
	if n, err := c.Read(raw); err != nil || n == 0 {
		return data, fmt.Errorf("could not get value: %w", err)
	}

	dec := newDecoder(bytes.NewReader(raw))
	dec.readCO2(&data.CO2)
	dec.readT(&data.T)
	dec.readP(&data.P)
	dec.readH(&data.H)
	dec.readBattery(&data.Battery)
	dec.readQuality(&data.Quality)
	dec.readInterval(&data.Interval)
	dec.readTime(&data.Time)

	if dec.err != nil {
		return data, fmt.Errorf("could not decode data sample: %w", dec.err)
	}

	return data, nil
}

func (dev *Device) NumData() (int, error) {
	c, err := dev.getCharByUUID(uuidReadTotalReadings)
	if err != nil {
		return 0, fmt.Errorf("could not get characteristic %q: %w", uuidReadTotalReadings, err)
	}

	raw := make([]byte, 255)
	if n, err := c.Read(raw); err != nil || n == 0 {
		return 0, fmt.Errorf("could not get value: %w", err)
	}

	return int(binary.LittleEndian.Uint16(raw)), nil
}

func (dev *Device) Since() (time.Duration, error) {
	c, err := dev.getCharByUUID(uuidReadSecondsSinceUpdate)
	if err != nil {
		return 0, fmt.Errorf("could not get characteristic %q: %w", uuidReadSecondsSinceUpdate, err)
	}

	raw := make([]byte, 255)
	if n, err := c.Read(raw); err != nil || n == 0 {
		return 0, fmt.Errorf("could not get value: %w", err)
	}

	var (
		ago time.Duration
		dec = newDecoder(bytes.NewReader(raw))
	)
	err = dec.readInterval(&ago)
	if err != nil {
		return 0, fmt.Errorf("could not decode interval value %q: %w", raw, err)
	}
	return ago, nil
}

func (dev *Device) Interval() (time.Duration, error) {
	c, err := dev.getCharByUUID(uuidReadInterval)
	if err != nil {
		return 0, fmt.Errorf("could not get characteristic %q: %w", uuidReadInterval, err)
	}

	raw := make([]byte, 255)
	if n, err := c.Read(raw); err != nil || n == 0 {
		return 0, fmt.Errorf("could not get value: %w", err)
	}

	var (
		ago time.Duration
		dec = newDecoder(bytes.NewReader(raw))
	)
	err = dec.readInterval(&ago)
	if err != nil {
		return 0, fmt.Errorf("could not decode interval value %q: %w", raw, err)
	}
	return ago, nil
}

// This doesnt really work right now.
func (dev *Device) ReadAll() ([]Data, error) {
	now := time.Now().UTC()
	ago, err := dev.Since()
	if err != nil {
		return nil, fmt.Errorf("could not get last measurement update: %w", err)
	}

	delta, err := dev.Interval()
	if err != nil {
		return nil, fmt.Errorf("could not get sampling: %w", err)
	}

	n, err := dev.NumData()
	if err != nil {
		return nil, fmt.Errorf("could not get total number of samples: %w", err)
	}
	out := make([]Data, n)
	for _, id := range []byte{paramT, paramH, paramP, paramCO2} {
		err = dev.readN(out, id)
		if err != nil {
			return nil, fmt.Errorf("could not read param=%d: %w", id, err)
		}
	}

	beg := now.Add(-ago - time.Duration(n-1)*delta)
	for i := range out {
		out[i].Battery = -1 // no battery information when fetching history.
		out[i].Quality = qualityFrom(out[i].CO2)
		out[i].Interval = delta
		out[i].Time = beg.Add(time.Duration(i) * delta)
	}

	return out, nil
}

// type btOptions = map[string]interface{}

// var opReqCmd = btOptions{
// 	"type": "request",
// }

func (dev *Device) readN(dst []Data, id byte) error {
	{
		cmd := []byte{
			0x82, 0x00, 0x00, 0x00, 0x01, 0x00, 0xff, 0xff,
		}
		cmd[1] = id
		binary.LittleEndian.PutUint16(cmd[4:], 0x0001)
		binary.LittleEndian.PutUint16(cmd[6:], 0xffff)

		c, err := dev.getCharByUUID(uuidWriteCmd)
		if err != nil {
			return fmt.Errorf("could not get characteristic %q: %w", uuidWriteCmd, err)
		}

		_, err = c.WriteWithoutResponse(cmd)
		if err != nil {
			return fmt.Errorf("could not write command: %w", err)
		}
	}

	c, err := dev.getCharByUUID(uuidReadTimeSeries)
	if err != nil {
		return fmt.Errorf("could not get characteristic %q: %w", uuidReadTimeSeries, err)
	}

	done := make(chan struct{})

	var errLoop error
	err = c.EnableNotifications(func(p []byte) {
		param := p[0]
		fmt.Println("param")
		fmt.Println(param)
		if param != id {
			errLoop = fmt.Errorf("invalid parameter: got=0x%x, want=0x%x", param, id)
			return
		}

		idx := int(binary.LittleEndian.Uint16(p[1:]) - 1)
		cnt := int(p[3])
		if cnt == 0 {
			close(done)
			return
		}
		max := min(idx+cnt, len(dst)) // a new sample may have appeared
		dec := newDecoder(bytes.NewReader(p[4:]))
		for i := idx; i < max; i++ {
			err := dec.readField(id, &dst[i])
			if err != nil {
				errLoop = fmt.Errorf("could not read param=%d, idx=%d: %w", id, i, err)
				return
			}
		}
		return
	})
	if err != nil {
		return fmt.Errorf("could not watch props: %w", err)
	}

	if errLoop != nil {
		return fmt.Errorf("could not read notified data: %w", errLoop)
	}

	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
