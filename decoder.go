// Copyright ©2022 The aranet4 Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package aranet4

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"
)

type decoder struct {
	r   io.Reader
	buf []byte
	err error
}

func newDecoder(r io.Reader) *decoder {
	return &decoder{
		r:   r,
		buf: make([]byte, 2),
	}
}

func (dec *decoder) load1() error {
	if dec.err != nil {
		return dec.err
	}
	_, dec.err = io.ReadFull(dec.r, dec.buf[:1])
	return dec.err
}

func (dec *decoder) load2() error {
	if dec.err != nil {
		return dec.err
	}
	_, dec.err = io.ReadFull(dec.r, dec.buf[:2])
	return dec.err
}

func (dec *decoder) readField(id byte, v *Data) error {
	if dec.err != nil {
		return dec.err
	}
	switch id {
	case paramT:
		return dec.readT(&v.T)
	case paramH:
		return dec.readH(&v.H)
	case paramP:
		return dec.readP(&v.P)
	case paramCO2:
		return dec.readCO2(&v.CO2)
	default:
		return fmt.Errorf("unknown field id=%d", id)
	}
}

func (dec *decoder) readCO2(v *int) error {
	err := dec.load2()
	if err != nil {
		return err
	}

	vv := binary.LittleEndian.Uint16(dec.buf)
	switch vv & 0x8000 {
	case 0x8000:
		return ErrNoData
	default:
		*v = int(vv)
	}

	return nil
}

func (dec *decoder) readT(v *float64) error {
	err := dec.load2()
	if err != nil {
		return err
	}

	vv := binary.LittleEndian.Uint16(dec.buf)
	switch {
	case vv == 0x4000:
		return ErrNoData
	case vv > 0x8000:
		*v = 0
	default:
		*v = float64(vv) / 20
	}

	return nil
}

func (dec *decoder) readP(v *float64) error {
	err := dec.load2()
	if err != nil {
		return err
	}

	vv := binary.LittleEndian.Uint16(dec.buf)
	switch {
	case vv&0x8000 == 0x8000:
		return ErrNoData
	default:
		*v = float64(vv) / 10
	}
	return nil
}

func (dec *decoder) readH(v *float64) error {
	err := dec.load1()
	if err != nil {
		return err
	}

	*v = float64(dec.buf[0])
	return nil
}

func (dec *decoder) readBattery(v *int) error {
	err := dec.load1()
	if err != nil {
		return err
	}
	*v = int(dec.buf[0])
	return nil
}

func (dec *decoder) readQuality(v *Quality) error {
	err := dec.load1()
	if err != nil {
		return err
	}
	*v = Quality(dec.buf[0])
	return nil
}

func (dec *decoder) readInterval(v *time.Duration) error {
	err := dec.load2()
	if err != nil {
		return err
	}

	*v = time.Duration(binary.LittleEndian.Uint16(dec.buf)) * time.Second
	return nil
}

func (dec *decoder) readTime(v *time.Time) error {
	err := dec.load2()
	if err != nil {
		return err
	}

	ago := time.Duration(binary.LittleEndian.Uint16(dec.buf)) * time.Second
	*v = time.Now().UTC().Add(-ago)
	return nil
}
