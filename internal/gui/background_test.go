package gui

import (
	"runtime"
	"sync"
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

// TestBackgroundTrackerMispairedDonePanics asserts the fail-fast of a unit
// of work that is finished without being started: like the negative counter
// panic of the sync.WaitGroup it replaced, an extra Done panics instead of
// silently driving the counter negative — a negative counter would let Wait
// return while background work is still running.
func TestBackgroundTrackerMispairedDonePanics(t *testing.T) {
	// a fresh tracker, so that the panic of the extra Done can not affect
	// the other test cases
	tracker := backgroundTracker{}
	tracker.cond = sync.NewCond(&tracker.mu)

	require.Panics(t, tracker.Done,
		"a Done without a matching Start must panic")

	// the panic must not leave the mutex locked: a balanced Start/Done
	// pair and a Wait still work on the tracker
	require.NotPanics(t, func() {
		tracker.Start()
		tracker.Done()
		tracker.Wait()
	}, "the tracker must stay usable after the panic")

	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	require.Zero(t, tracker.count,
		"a balanced Start/Done pair leaves no work counted")
}
