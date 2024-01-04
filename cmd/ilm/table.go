// Copyright (c) 2022 MinIO, Inc.
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

package ilm

import (
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/minio/minio-go/v7/pkg/lifecycle"
)

// Table interface provides methods when implemented allows a []T to be rendered
// as a table.
type Table interface {
	Len() int
	Title() string
	Rows() []table.Row
	ColumnHeaders() table.Row
}

// LsFilter enumerates the 3 possible ilm-ls filter options.
type LsFilter uint8

const (
	// None - no filter
	None LsFilter = iota
	// ExpiryOnly - filter expiration actions across rules
	ExpiryOnly
	// TransitionOnly - filter transition actions across rules
	TransitionOnly
)

// Apply applies f on rules and filters lifecycle rules matching it
func (f LsFilter) Apply(rules []lifecycle.Rule) []lifecycle.Rule {
	check := func(rule lifecycle.Rule) bool {
		switch f {
		case ExpiryOnly:
			return !rule.Expiration.IsNull() || !rule.NoncurrentVersionExpiration.IsDaysNull() ||
				rule.NoncurrentVersionExpiration.NewerNoncurrentVersions > 0
		case TransitionOnly:
			return !rule.Transition.IsNull() || !rule.NoncurrentVersionTransition.IsStorageClassEmpty()
		}
		return true
	}

	var n int
	for _, rule := range rules {
		if check(rule) {
			rules[n] = rule
			n++
		}
	}
	rules = rules[:n]
	return rules
}

type expirationCurrentRow struct {
	ID              string
	Status          string
	Prefix          string
	Tags            string
	Days            int
	ExpireDelMarker bool
}

type expirationCurrentTable []expirationCurrentRow

func (e expirationCurrentTable) Len() int {
	return len(e)
}

func (e expirationCurrentTable) Title() string {
	return "Expiration for latest version (Expiration)"
}

func (e expirationCurrentTable) Rows() (rows []table.Row) {
	for _, row := range e {
		if row.Prefix == "" {
			row.Prefix = "-"
		}
		if row.Tags == "" {
			row.Tags = "-"
		}
		rows = append(rows, table.Row{row.ID, row.Status, row.Prefix, row.Tags, row.Days, row.ExpireDelMarker})
	}
	return rows
}

func (e expirationCurrentTable) ColumnHeaders() (headers table.Row) {
	return table.Row{"ID", "Status", "Prefix", "Tags", "Days to Expire", "Expire DeleteMarker"}
}

type expirationNoncurrentTable []expirationNoncurrentRow

type expirationNoncurrentRow struct {
	ID           string
	Status       string
	Prefix       string
	Tags         string
	Days         int
	KeepVersions int
}

func (e expirationNoncurrentTable) Len() int {
	return len(e)
}

func (e expirationNoncurrentTable) Title() string {
	return "Expiration for older versions (NoncurrentVersionExpiration)"
}

func (e expirationNoncurrentTable) Rows() (rows []table.Row) {
	for _, row := range e {
		if row.Prefix == "" {
			row.Prefix = "-"
		}
		if row.Tags == "" {
			row.Tags = "-"
		}
		rows = append(rows, table.Row{row.ID, row.Status, row.Prefix, row.Tags, row.Days, row.KeepVersions})
	}
	return rows
}

func (e expirationNoncurrentTable) ColumnHeaders() (headers table.Row) {
	return table.Row{"ID", "Status", "Prefix", "Tags", "Days to Expire", "Keep Versions"}
}

type tierCurrentTable []tierCurrentRow

type tierCurrentRow struct {
	ID     string
	Status string
	Prefix string
	Tags   string
	Days   int
	Tier   string
}

func (t tierCurrentTable) Len() int {
	return len(t)
}

func (t tierCurrentTable) Title() string {
	return "Transition for latest version (Transition)"
}

func (t tierCurrentTable) ColumnHeaders() (headers table.Row) {
	return table.Row{"ID", "Status", "Prefix", "Tags", "Days to Tier", "Tier"}
}

func (t tierCurrentTable) Rows() (rows []table.Row) {
	for _, row := range t {
		if row.Prefix == "" {
			row.Prefix = "-"
		}
		if row.Tags == "" {
			row.Tags = "-"
		}
		rows = append(rows, table.Row{row.ID, row.Status, row.Prefix, row.Tags, row.Days, row.Tier})
	}
	return rows
}

type (
	tierNoncurrentTable []tierNoncurrentRow
	tierNoncurrentRow   tierCurrentRow
)

func (t tierNoncurrentTable) Len() int {
	return len(t)
}

func (t tierNoncurrentTable) Title() string {
	return "Transition for older versions (NoncurrentVersionTransition)"
}

func (t tierNoncurrentTable) ColumnHeaders() table.Row {
	return table.Row{"ID", "Status", "Prefix", "Tags", "Days to Tier", "Tier"}
}

func (t tierNoncurrentTable) Rows() (rows []table.Row) {
	for _, row := range t {
		if row.Prefix == "" {
			row.Prefix = "-"
		}
		if row.Tags == "" {
			row.Tags = "-"
		}
		rows = append(rows, table.Row{row.ID, row.Status, row.Prefix, row.Tags, row.Days, row.Tier})
	}
	return rows
}
