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

import "time"

// HealthReportInfo - interface to be implemented by health report schema struct
type HealthReportInfo interface {
	GetTimestamp() time.Time
	GetStatus() string
	GetError() string
	message
}

// HealthReportHeader - Header of the subnet health report
// expected to generate JSON output of the form
// {"subnet":{"health":{"version":"v1"}}}
type HealthReportHeader struct {
	Subnet Health `json:"subnet"`
}

// Health - intermediate struct for subnet health header
// Just used to achieve the JSON structure we want
type Health struct {
	Health SchemaVersion `json:"health"`
}

// SchemaVersion - version of the health report schema
type SchemaVersion struct {
	Version string `json:"version"`
}
