// Copyright ©2022 The aranet4 Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"sort"
	"time"

	"go.etcd.io/bbolt"
	"sbinet.org/x/aranet4"
)

const (
	timeResolution int64 = 5 // seconds
)

func ltApprox(a, b aranet4.Data) bool {
	at := a.Time.UTC().Unix()
	bt := b.Time.UTC().Unix()
	if abs(at-bt) < timeResolution {
		return false
	}
	return at < bt
}

func abs(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

var (
	bucketData = []byte("aranet4")
)

func (srv *server) init() error {
	srv.mu.Lock()
	defer srv.mu.Unlock()

	err := srv.db.Update(func(tx *bbolt.Tx) error {
		data, err := tx.CreateBucketIfNotExists(bucketData)
		if err != nil {
			return fmt.Errorf("could not create %q bucket: %w", bucketData, err)
		}
		if data == nil {
			return fmt.Errorf("could not create %q bucket", bucketData)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("could not setup aranet4 db buckets: %w", err)
	}

	err = srv.db.View(func(tx *bbolt.Tx) error {
		bkt := tx.Bucket(bucketData)
		if bkt == nil {
			return fmt.Errorf("could not find %q bucket", bucketData)
		}
		return bkt.ForEach(func(k, v []byte) error {
			id := int64(binary.LittleEndian.Uint64(k))
			if id-srv.last.Time.UTC().Unix() > timeResolution {
				return unmarshalBinary(&srv.last, v)
			}
			return nil
		})
	})
	if err != nil {
		return fmt.Errorf("could not find last data sample: %w", err)
	}

	data, err := srv.rows()
	if err != nil {
		return fmt.Errorf("could not read data from db: %w", err)
	}

	err = srv.plot(data)
	if err != nil {
		return fmt.Errorf("could not generate initial plots: %w", err)
	}

	return nil
}

func (srv *server) update() error {
	srv.mu.Lock()
	defer srv.mu.Unlock()

	data, err := srv.fetchRows()
	if err != nil {
		return err
	}

	err = srv.write(data)
	if err != nil {
		return err
	}

	data, err = srv.rows()
	if err != nil {
		return err
	}

	return srv.plot(data)
}

func (srv *server) rows() ([]aranet4.Data, error) {
	var rows []aranet4.Data
	err := srv.db.View(func(tx *bbolt.Tx) error {
		bkt := tx.Bucket(bucketData)
		if bkt == nil {
			return fmt.Errorf("could not find %q bucket", bucketData)
		}
		return bkt.ForEach(func(k, v []byte) error {
			var (
				row aranet4.Data
				err = unmarshalBinary(&row, v)
			)
			if err != nil {
				return err
			}
			rows = append(rows, row)
			return nil
		})
	})
	if err != nil {
		return nil, fmt.Errorf("could not read rows: %w", err)
	}

	sort.Slice(rows, func(i, j int) bool {
		return ltApprox(rows[i], rows[j])
	})

	return rows, nil
}

func (srv *server) write(vs []aranet4.Data) error {
	if len(vs) == 0 {
		return nil
	}

	sort.Slice(vs, func(i, j int) bool {
		return ltApprox(vs[i], vs[j])
	})

	// consolidate data-from-sensor and time-series from db.
	idx := len(vs)
	for i, v := range vs {
		if ltApprox(srv.last, v) {
			idx = i
			break
		}
	}
	vs = vs[idx:]
	if len(vs) == 0 {
		return nil
	}

	plural := ""
	if len(vs) > 1 {
		plural = "s"
	}
	log.Printf("writing %d new sample%s to db...", len(vs), plural)
	err := srv.db.Update(func(tx *bbolt.Tx) error {
		bkt := tx.Bucket(bucketData)
		if bkt == nil {
			return fmt.Errorf("could not access %q bucket", bucketData)
		}

		for _, v := range vs {
			var (
				id  = make([]byte, 8)
				buf = make([]byte, dataSize)
			)
			unix := v.Time.UTC().Unix()
			binary.LittleEndian.PutUint64(id, uint64(unix))
			err := marshalBinary(v, buf)
			if err != nil {
				return fmt.Errorf("could not marshal sample %v: %w", v, err)
			}

			err = bkt.Put(id, buf)
			if err != nil {
				return fmt.Errorf("could not store sample %v: %w", v, err)
			}
			if ltApprox(srv.last, v) {
				srv.last = v
				srv.last.Quality = qualityFrom(srv.last.CO2)
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("could not write data slice to db: %w", err)
	}
	return nil
}

const dataSize = 17

func qualityFrom(co2 int) aranet4.Quality {
	switch {
	case co2 < 1000:
		return 1
	case co2 < 1400:
		return 2
	default:
		return 3
	}
}

func unmarshalBinary(data *aranet4.Data, p []byte) error {
	if len(p) != dataSize {
		return io.ErrShortBuffer
	}
	data.Time = time.Unix(int64(binary.LittleEndian.Uint64(p)), 0).UTC()
	data.H = float64(p[8])
	data.P = float64(binary.LittleEndian.Uint16(p[9:])) / 10
	data.T = float64(binary.LittleEndian.Uint16(p[11:])) / 100
	data.CO2 = int(binary.LittleEndian.Uint16(p[13:]))
	data.Battery = int(p[15])
	data.Quality = qualityFrom(data.CO2)
	data.Interval = time.Duration(p[16]) * time.Minute
	return nil
}

func marshalBinary(data aranet4.Data, p []byte) error {
	if len(p) != dataSize {
		return io.ErrShortBuffer
	}
	binary.LittleEndian.PutUint64(p[0:], uint64(data.Time.UTC().Unix()))
	p[8] = uint8(data.H)
	binary.LittleEndian.PutUint16(p[9:], uint16(data.P*10))
	binary.LittleEndian.PutUint16(p[11:], uint16(data.T*100))
	binary.LittleEndian.PutUint16(p[13:], uint16(data.CO2))
	p[15] = uint8(data.Battery)
	p[16] = uint8(data.Interval.Minutes())
	return nil
}

func (srv *server) fetchRows() ([]aranet4.Data, error) {
	dev, err := aranet4.New(srv.addr)
	if err != nil {
		return nil, fmt.Errorf("could not create aranet4 client: %w", err)
	}
	defer dev.Close()

	return dev.ReadAll()
}

func (srv *server) interval() (time.Duration, error) {
	dev, err := aranet4.New(srv.addr)
	if err != nil {
		return 0, fmt.Errorf("could not create aranet4 client: %w", err)
	}
	defer dev.Close()

	return dev.Interval()
}
