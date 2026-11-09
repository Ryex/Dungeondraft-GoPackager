package gui

import (
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestBackgroundTrackerConcurrentStartAndWait documents how the background
// work of the app is tracked: a worker goroutine starts and finishes units
// of work while this goroutine waits for the work to be done.
//
// This is what a sync.WaitGroup can not do: its Add may not be called
// concurrently with its Wait. backgroundTracker allows the worker to start
// the next unit of work while this goroutine waits for the running ones.
func TestBackgroundTrackerConcurrentStartAndWait(t *testing.T) {
	a := NewApp()

	const rounds = 500

	started := make(chan struct{})
	workerDone := make(chan struct{})
	go func() {
		close(started)
		defer close(workerDone)
		for i := 0; i < rounds; i++ {
			a.bgWork.Start()
			// keep the unit of work open for a moment, so that the
			// waiter of the main goroutine really has to wait for it
			runtime.Gosched()
			a.bgWork.Done()
		}
	}()
	<-started

	for i := 0; i < rounds; i++ {
		a.bgWork.Wait()
	}
	<-workerDone

	// Wait may only return when no unit of work is counted: while work is
	// counted it has to block, it returns once the work is done.
	a.bgWork.Start()
	blocked := make(chan struct{})
	go func() {
		a.bgWork.Wait()
		close(blocked)
	}()
	select {
	case <-blocked:
		t.Fatal("Wait must not return while background work is running")
	case <-time.After(50 * time.Millisecond):
	}
	a.bgWork.Done()
	select {
	case <-blocked:
	case <-time.After(10 * time.Second):
		t.Fatal("Wait must return once the background work is done")
	}

	// no unit of work is left once Wait has returned
	a.bgWork.mu.Lock()
	defer a.bgWork.mu.Unlock()
	require.Zero(t, a.bgWork.count,
		"no background work may be counted after Wait returned")
}
