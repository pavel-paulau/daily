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

func generateRows(reports []Report) map[string][][]string {
	rows := map[string][][]string{
		"Failed": {},
		"Passed": {},
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

func renderReport(writer io.Writer, reports []Report) {
	table := newTable(writer)
	rows := generateRows(reports)

	table.AppendBulk(rows["Failed"])
	if len(rows["Failed"]) > 0 {
		table.Append(newSeparator())
	}
	table.AppendBulk(rows["Passed"])

	table.Render()
}
