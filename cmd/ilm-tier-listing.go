package cmd

import "fmt"

type tabulator interface {
	ToRow(i int, lengths []int) []string // returns row representation i-th element in collection, modifies lengths s.t it contains maximum column widths incl this row.
	NumRows() int
	NumCols() int
	EmptyMessage() string
}

func toTable(tbl tabulator) string {
	if tbl.NumRows() == 0 {
		return tbl.EmptyMessage()
	}

	const tableSeparator = "|"
	rows, cols := getRowsAndCols(tbl)
	table := newPrettyTable(tableSeparator, cols...)
	var contents string
	for _, row := range rows {
		contents += fmt.Sprintf("%s\n", table.buildRow(row...))
	}
	return contents
}

func getRowsAndCols(tbl tabulator) ([][]string, []Field) {
	rows := make([][]string, 0, tbl.NumRows()+1)
	lengths := make([]int, tbl.NumCols())
	rows = append(rows, tbl.ToRow(-1, lengths))
	for i := 0; i < tbl.NumRows(); i++ {
		rows = append(rows, tbl.ToRow(i, lengths))
	}
	cols := make([]Field, tbl.NumCols())
	for i, hdr := range rows[0] {
		cols[i] = Field{
			colorTheme: hdr,
			maxLen:     lengths[i] + 2,
		}
	}
	return rows, cols
}
