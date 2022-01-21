// Copyright ©2022 The aranet4 Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"reflect"
	"testing"
	"time"

	"sbinet.org/x/aranet4"
)

func TestRWData(t *testing.T) {
	want := aranet4.Data{
		H:        100,
		P:        1000,
		T:        100.12,
		CO2:      2000,
		Battery:  100,
		Quality:  3,
		Interval: 5 * time.Minute,
		Time:     time.Date(2006, time.January, 2, 15, 4, 5, 0, time.UTC),
	}

	buf := make([]byte, dataSize)
	err := marshalBinary(want, buf)
	if err != nil {
		t.Fatalf("could not marshal binary: %+v", err)
	}

	var got aranet4.Data
	err = unmarshalBinary(&got, buf)
	if err != nil {
		t.Fatalf("could not unmarshal binary: %+v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("invalid roundtrip:\ngot:\n%vwant:\n%v", got, want)
	}
}
