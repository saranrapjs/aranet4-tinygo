// Copyright ©2022 The aranet4 Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package aranet4

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/muka/go-bluetooth/bluez/profile/adapter"
	"github.com/muka/go-bluetooth/bluez/profile/device"
)

type Device struct {
	addr string
	dev  *device.Device1
}

func New(addr string) (*Device, error) {
	ad, err := adapter.GetDefaultAdapter()
	if err != nil {
		return nil, fmt.Errorf("could not find default adapter: %w", err)
	}
	powered, err := ad.GetPowered()
	if err != nil {
		return nil, fmt.Errorf("could not check default adapter power: %w", err)
	}
	if !powered {
		err = ad.SetPowered(true)
		if err != nil {
			return nil, fmt.Errorf("could not set default adapter power: %w", err)
		}
	}

	dev, err := ad.GetDeviceByAddress(addr)
	if err != nil {
		return nil, fmt.Errorf("could not find device %q: %w", addr, err)
	}

	err = dev.Connect()
	if err != nil {
		dev.Close()
		return nil, fmt.Errorf("could not connect to %q: %w", addr, err)
	}

	return &Device{addr: addr, dev: dev}, nil
}

func (dev *Device) Close() error {
	if dev.dev == nil {
		return nil
	}
	defer func() {
		dev.dev.Close()
		dev.dev = nil
	}()

	err := dev.dev.Disconnect()
	if err != nil {
		return fmt.Errorf("could not disconnect: %w", err)
	}
	return nil
}

func (dev *Device) Name() (string, error) {
	return dev.dev.GetName()
}

func (dev *Device) Version() (string, error) {
	c, err := dev.dev.GetCharByUUID(uuidCommonReadSWRevision)
	if err != nil {
		return "", fmt.Errorf("could not get characteristic %q: %w", uuidCommonReadSWRevision, err)
	}
	defer c.Close()

	raw, err := c.ReadValue(nil)
	if err != nil {
		return "", fmt.Errorf("could not read device name: %w", err)
	}
	return string(raw), nil
}

func (dev *Device) Read() (Data, error) {
	var data Data

	c, err := dev.dev.GetCharByUUID(uuidReadAll)
	if err != nil {
		return data, fmt.Errorf("could not get characteristic %q: %w", uuidReadAll, err)
	}
	defer c.Close()

	raw, err := c.ReadValue(nil)
	if err != nil {
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
	c, err := dev.dev.GetCharByUUID(uuidReadTotalReadings)
	if err != nil {
		return 0, fmt.Errorf("could not get characteristic %q: %w", uuidReadTotalReadings, err)
	}
	defer c.Close()

	raw, err := c.ReadValue(nil)
	if err != nil {
		return 0, fmt.Errorf("could not get value: %w", err)
	}

	return int(binary.LittleEndian.Uint16(raw)), nil
}

func (dev *Device) Since() (time.Duration, error) {
	c, err := dev.dev.GetCharByUUID(uuidReadSecondsSinceUpdate)
	if err != nil {
		return 0, fmt.Errorf("could not get characteristic %q: %w", uuidReadSecondsSinceUpdate, err)
	}
	defer c.Close()

	raw, err := c.ReadValue(nil)
	if err != nil {
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
	c, err := dev.dev.GetCharByUUID(uuidReadInterval)
	if err != nil {
		return 0, fmt.Errorf("could not get characteristic %q: %w", uuidReadInterval, err)
	}
	defer c.Close()

	raw, err := c.ReadValue(nil)
	if err != nil {
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
		out[i].Time = beg.Add(time.Duration(i) * delta)
		out[i].Interval = delta
	}

	return out, nil
}

func (dev *Device) readN(dst []Data, id byte) error {
	{
		cmd := []byte{
			0x82, 0x00, 0x00, 0x00, 0x01, 0x00, 0xff, 0xff,
		}
		cmd[1] = id
		binary.LittleEndian.PutUint16(cmd[4:], 0x0001)
		binary.LittleEndian.PutUint16(cmd[6:], 0xffff)

		c, err := dev.dev.GetCharByUUID(uuidWriteCmd)
		if err != nil {
			return fmt.Errorf("could not get characteristic %q: %w", uuidWriteCmd, err)
		}
		defer c.Close()

		err = c.WriteValue(cmd, nil)
		if err != nil {
			return fmt.Errorf("could not write command: %w", err)
		}
	}

	c, err := dev.dev.GetCharByUUID(uuidReadTimeSeries)
	if err != nil {
		return fmt.Errorf("could not get characteristic %q: %w", uuidReadTimeSeries, err)
	}

	ch, err := c.WatchProperties()
	if err != nil {
		return fmt.Errorf("could not watch props: %w", err)
	}

	err = c.StartNotify()
	if err != nil {
		return fmt.Errorf("could not start notify: %w", err)
	}

	done := make(chan struct{})
	cbk := func(p []byte) error {
		param := p[0]
		if param != id {
			return fmt.Errorf("invalid parameter: got=0x%x, want=0x%x", param, id)
		}

		idx := int(binary.LittleEndian.Uint16(p[1:]) - 1)
		cnt := int(p[3])
		if cnt == 0 {
			close(done)
			return io.EOF
		}
		max := min(idx+cnt, len(dst)) // a new sample may have appeared
		dec := newDecoder(bytes.NewReader(p[4:]))
		for i := idx; i < max; i++ {
			err := dec.readField(id, &dst[i])
			if err != nil {
				return fmt.Errorf("could not read param=%d, idx=%d: %w", id, i, err)
			}
		}
		return nil
	}

	var errLoop error
	go func() {
		const iface = "org.bluez.GattCharacteristic1"
		for v := range ch {
			if v == nil {
				return
			}
			if v.Interface == iface && v.Name == "Value" {
				err := cbk(v.Value.([]byte))
				if err != nil {
					if !errors.Is(err, io.EOF) {
						errLoop = err
					}
				}
			}
		}
	}()
	<-done

	err = c.UnwatchProperties(ch)
	if err != nil {
		return fmt.Errorf("could not unwatch props: %w", err)
	}

	err = c.StopNotify()
	if err != nil {
		return fmt.Errorf("could not stop-notify: %w", err)
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
