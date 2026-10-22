package gui

import "sync"

// progressReporter forwards the progress of a long running task to the
// interface.
//
// The progress of a task is reported for every file of the package, and
// every report queues a function on the Fyne goroutine. Without a throttle,
// the queue of the goroutine would hold a function for every file of the
// package, which slows the interface down (and blocks it) for a package with
// many files.
type progressReporter struct {
	// callback is called with the progress of the task.
	callback func(p float64)

	mu sync.Mutex
	// last is the progress that was reported last.
	last float64
}

// newProgressReporter creates a reporter that forwards the progress to the
// given callback, but only when it changed by at least a percent.
func newProgressReporter(callback func(p float64)) *progressReporter {
	return &progressReporter{callback: callback}
}

// Report forwards the progress to the callback of the reporter, if it
// changed by at least a percent since the last report.
func (r *progressReporter) Report(p float64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if int(p*100) == int(r.last*100) {
		return
	}
	r.last = p

	if r.callback != nil {
		r.callback(p)
	}
}
