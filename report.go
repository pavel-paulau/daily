package main

import (
	"fmt"
	"io"

	"github.com/dustin/go-humanize"
	"github.com/olekukonko/tablewriter"
)

func evalStatus(delta, threshold float64) string {
	if threshold < 0 && delta < threshold {
		return "Failed"
	}
	if threshold > 0 && delta > threshold {
		return "Failed"
	}
	return "Passed"
}

func newTable(writer io.Writer) *tablewriter.Table {
	table := tablewriter.NewWriter(writer)
	table.SetAutoWrapText(false)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetHeader([]string{"Component", "Test Case", "Metric", "Result", "Delta", "Status"})

	return table
}

func newSeparator() []string {
	return []string{"", "", "", "", "", ""}
}

func addMissing(rows map[string][][]string, tc TestCase) {
	row := []string{tc.Component, tc.TestCase, tc.Metric, "N/A", "N/A", "Missing"}
	rows["Missing"] = append(rows["Missing"], row)
}

func generateRows(reports []Report, testCases []TestCase) map[string][][]string {
	rows := map[string][][]string{
		"Missing": {},
		"Failed": {},
		"Passed": {},
	}

	i := 0
	for _, tc := range testCases {
		if i >= len(reports) {
			addMissing(rows, tc)
			continue
		}
		r := reports[i]
		if tc.Component != tc.Component || tc.TestCase != r.TestCase {
			addMissing(rows, tc)
			continue
		}
		i++
	}

	for _, r := range reports {
		result := humanize.Commaf(r.Value)

		deltaF := 100 * (r.Value/r.MovingAverage - 1)
		delta := fmt.Sprintf("%.1f", deltaF)
		if deltaF > 0 {
			delta = "+" + delta
		}

		status := evalStatus(deltaF, float64(r.Threshold))

		row := []string{r.Component, r.TestCase, r.Metric, result, delta, status}

		rows[status] = append(rows[status], row)
	}
	return rows
}

func renderReport(writer io.Writer, reports []Report, testCases []TestCase) {
	table := newTable(writer)
	rows := generateRows(reports, testCases)

	for _, row := range rows {
		table.AppendBulk(row)
		if len(row) > 0 {
			table.Append(newSeparator())
		}
	}

	table.Render()
}
