// Copyright (c) 2015-2021 MinIO, Inc.
//
// This file is part of MinIO Object Storage stack
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package cmd

import (
	"fmt"
	"math"
	"time"
)

// humanizedDuration container to capture humanized time.
type humanizedDuration struct {
	Days    int64 `json:"days,omitempty"`
	Hours   int64 `json:"hours,omitempty"`
	Minutes int64 `json:"minutes,omitempty"`
	Seconds int64 `json:"seconds,omitempty"`
}

// StringShort() humanizes humanizedDuration to human readable short format.
// This does not print at seconds.
func (r humanizedDuration) StringShort() string {
	switch {
	case r.Days == 0 && r.Hours == 0 && r.Minutes == 0:
		return fmt.Sprintf("%d seconds", r.Seconds)
	case r.Days == 0 && r.Hours == 0:
		return fmt.Sprintf("%d minutes", r.Minutes)
	case r.Days == 0:
		return fmt.Sprintf("%d hours %d minutes", r.Hours, r.Minutes)
	case r.Days <= 2:
		return fmt.Sprintf("%d days, %d hours", r.Days, r.Hours)
	default:
		return fmt.Sprintf("%d days", r.Days)
	}
}

// String() humanizes humanizedDuration to human readable,
func (r humanizedDuration) String() string {
	if r.Days == 0 && r.Hours == 0 && r.Minutes == 0 {
		return fmt.Sprintf("%d seconds", r.Seconds)
	}
	if r.Days == 0 && r.Hours == 0 {
		return fmt.Sprintf("%d minutes %d seconds", r.Minutes, r.Seconds)
	}
	if r.Days == 0 {
		return fmt.Sprintf("%d hours %d minutes %d seconds", r.Hours, r.Minutes, r.Seconds)
	}
	return fmt.Sprintf("%d days %d hours %d minutes %d seconds", r.Days, r.Hours, r.Minutes, r.Seconds)
}

// timeDurationToHumanizedDuration convert golang time.Duration to a custom more readable humanizedDuration.
func timeDurationToHumanizedDuration(duration time.Duration) humanizedDuration {
	r := humanizedDuration{}
	if duration.Seconds() < 60.0 {
		r.Seconds = int64(duration.Seconds())
		return r
	}
	if duration.Minutes() < 60.0 {
		remainingSeconds := math.Mod(duration.Seconds(), 60)
		r.Seconds = int64(remainingSeconds)
		r.Minutes = int64(duration.Minutes())
		return r
	}
	if duration.Hours() < 24.0 {
		remainingMinutes := math.Mod(duration.Minutes(), 60)
		remainingSeconds := math.Mod(duration.Seconds(), 60)
		r.Seconds = int64(remainingSeconds)
		r.Minutes = int64(remainingMinutes)
		r.Hours = int64(duration.Hours())
		return r
	}
	remainingHours := math.Mod(duration.Hours(), 24)
	remainingMinutes := math.Mod(duration.Minutes(), 60)
	remainingSeconds := math.Mod(duration.Seconds(), 60)
	r.Hours = int64(remainingHours)
	r.Minutes = int64(remainingMinutes)
	r.Seconds = int64(remainingSeconds)
	r.Days = int64(duration.Hours() / 24)
	return r
}
