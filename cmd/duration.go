// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the https://go.dev/LICENSE file.
//
// Code is borrowed directly from pkg/time.ParseDuration under
// Go's BSD-style license, extended with changes to support
// days, weeks and year as time Duration units.

// Package cmd provides extended time.Duration implementation
package cmd

import (
	"errors"
	"strings"
	"time"
)

// Duration is a standard unit of time.
type Duration time.Duration

// Days returns the duration as a floating point number of days.
func (d Duration) Days() float64 {
	hour := d / Hour
	nsec := d % Hour
	return float64(hour) + float64(nsec)*(1e-9/60/60/24)
}

// Standard unit of time.
var (
	Nanosecond  = Duration(time.Nanosecond)
	Microsecond = Duration(time.Microsecond)
	Millisecond = Duration(time.Millisecond)
	Second      = Duration(time.Second)
	Minute      = Duration(time.Minute)
	Hour        = Duration(time.Hour)
	Day         = Hour * 24
	Week        = Day * 7
	Fortnight   = Week * 2
	Month       = Day * 30    // Approximation
	Year        = Day * 365   // Approximation
	Decade      = Year * 10   // Approximation
	Century     = Year * 100  // Approximation
	Millennium  = Year * 1000 // Approximation
)

// leadingInt consumes the leading [0-9]* from s.
// this function is directly copied from
// https://cs.opensource.google/go/go/+/refs/tags/go1.18.3:src/time/format.go;drc=55590f3a2b89f001bcadf0df6eb2dde62618302b;l=1455
// only changes the "error string that's being returned"
func leadingInt(s string) (x int64, rem string, err error) {
	errLeadingInt := errors.New("bad [0-9]*")

	i := 0
	for ; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			break
		}
		if x > (1<<63-1)/10 {
			// overflow
			return 0, "", errLeadingInt
		}
		x = x*10 + int64(c) - '0'
		if x < 0 {
			// overflow
			return 0, "", errLeadingInt
		}
	}
	return x, s[i:], nil
}

// unitMap defines 'unit' to 'actual-value'
// translates for calculating duration.
var unitMap = map[string]int64{
	"ns": int64(Nanosecond),
	"us": int64(Microsecond),
	"µs": int64(Microsecond), // U+00B5 = micro symbol
	"μs": int64(Microsecond), // U+03BC = Greek letter mu
	"ms": int64(Millisecond),
	"s":  int64(Second),
	"m":  int64(Minute),
	"h":  int64(Hour),
	"d":  int64(Day),
	"w":  int64(Week),
	"y":  int64(Year), // Approximation
}

// ParseDuration parses a duration string.
// This implementation is similar to Go's
// https://cs.opensource.google/go/go/+/refs/tags/go1.18.3:src/time/format.go;l=1546;bpv=1;bpt=1
// extends it to parse days, weeks and years..
//
// Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h", "d", "w", "y".
func ParseDuration(s string) (Duration, error) {
	if strings.TrimSpace(s) == "" {
		return 0, errors.New("invalid empty duration")
	}

	// [-+]?([0-9]*(\.[0-9]*)?[a-z]+)+
	orig := s
	var d int64
	neg := false

	// Consume [-+]?
	if s != "" {
		c := s[0]
		if c == '-' || c == '+' {
			neg = c == '-'
			s = s[1:]
		}
	}
	// Special case: if all that is left is "0", this is zero.
	if s == "0" {
		return 0, nil
	}
	if s == "" {
		return 0, errors.New("invalid duration " + orig)
	}
	for s != "" {
		var (
			v, f  int64       // integers before, after decimal point
			scale float64 = 1 // value = v + f/scale
		)

		var err error

		// The next character must be [0-9.]
		if s[0] != '.' && ('0' > s[0] || s[0] > '9') {
			return 0, errors.New("invalid duration " + orig)
		}
		// Consume [0-9]*
		pl := len(s)
		v, s, err = leadingInt(s)
		if err != nil {
			return 0, errors.New("invalid duration " + orig)
		}
		pre := pl != len(s) // whether we consumed anything before a period

		// Consume (\.[0-9]*)?
		post := false
		if s != "" && s[0] == '.' {
			s = s[1:]
			pl := len(s)
			f, s, err = leadingInt(s)
			if err != nil {
				return 0, errors.New("invalid duration " + orig)
			}
			for n := pl - len(s); n > 0; n-- {
				scale *= 10
			}
			post = pl != len(s)
		}
		if !pre && !post {
			// no digits (e.g. ".s" or "-.s")
			return 0, errors.New("invalid duration " + orig)
		}

		// Consume unit.
		i := 0
		for ; i < len(s); i++ {
			c := s[i]
			if c == '.' || '0' <= c && c <= '9' {
				break
			}
		}
		if i == 0 {
			return 0, errors.New("missing unit in duration " + orig)
		}
		u := s[:i]
		s = s[i:]
		unit, ok := unitMap[u]
		if !ok {
			return 0, errors.New("unknown unit " + u + " in duration " + orig)
		}
		if v > (1<<63-1)/unit {
			// overflow
			return 0, errors.New("invalid duration " + orig)
		}
		v *= unit
		if f > 0 {
			// float64 is needed to be nanosecond accurate for fractions of hours.
			// v >= 0 && (f*unit/scale) <= 3.6e+12 (ns/h, h is the largest unit)
			v += int64(float64(f) * (float64(unit) / scale))
			if v < 0 {
				// overflow
				return 0, errors.New("invalid duration " + orig)
			}
		}
		d += v
		if d < 0 {
			// overflow
			return 0, errors.New("invalid duration " + orig)
		}
	}

	if neg {
		d = -d
	}
	return Duration(d), nil
}
