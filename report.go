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

func renderReport(writer io.Writer, b1, b2 string, reports []Report) {
	table := tablewriter.NewWriter(writer)
	table.SetHeader([]string{"Component", "Test Case", "Metric", b1, b2, "Delta", "Status"})
	table.SetAutoWrapText(false)
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	for _, r := range reports {
		v1 := humanize.Commaf(r.Results[0].Value)
		v2 := humanize.Commaf(r.Results[1].Value)

		deltaF := 100 * (r.Results[1].Value/r.Results[0].Value - 1)
		delta := fmt.Sprintf("%.1f", deltaF)
		if deltaF > 0 {
			delta = "+" + delta
		}

		status := evalStatus(deltaF, float64(r.Threshold))

		row := []string{r.Component, r.TestCase, r.Metric, v1, v2, delta, status}

		table.Append(row)
	}
	table.Render()
}
