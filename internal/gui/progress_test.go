package gui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestProgressReporterThrottlesTheReports verifies that the progress of a
// task is only reported when it changed by at least a percent.
//
// The progress of a task is reported for every file of the package, and every
// report queues a function on the Fyne goroutine. Without the throttle, the
// queue of the goroutine would hold a function for every file of the
// package, which slows the interface down (and blocks it) for a package with
// many files.
func TestProgressReporterThrottlesTheReports(t *testing.T) {
	reports := 0
	reporter := newProgressReporter(func(p float64) {
		reports++
	})

	for i := range 10000 {
		reporter.Report(float64(i) / 10000)
	}

	assert.Greater(t, reports, 0, "the progress must be reported")
	assert.LessOrEqual(t, reports, 101,
		"the progress must only be reported when it changed by at least a percent")
}

// TestProgressReporterReportsTheEndOfTheTask verifies that the progress of a
// task that is done is reported.
func TestProgressReporterReportsTheEndOfTheTask(t *testing.T) {
	reports := []float64{}
	reporter := newProgressReporter(func(p float64) {
		reports = append(reports, p)
	})

	reporter.Report(0.5)
	reporter.Report(0.5)
	reporter.Report(1)

	assert.Equal(t, []float64{0.5, 1}, reports,
		"the progress must be reported when it changed and at the end of the task")
}
