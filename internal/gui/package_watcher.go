package gui

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/ryex/dungeondraft-gopackager/internal/utils"
	"github.com/ryex/dungeondraft-gopackager/pkg/structures"

	log "github.com/sirupsen/logrus"
)

// defaultPackageWatcherDebounce is the time the watcher waits for further
// events, before it reports the changed paths to the app.
//
// The events of a single change come in bursts (e.g. an editor that saves a
// file writes a temporary file and renames it), so the changes are collected
// for a moment, to update the package only once.
const defaultPackageWatcherDebounce = 1 * time.Second

// packageWatcher watches the unpacked folder of a package for changes.
//
// Every changed path is recorded in a queue, in which it is stored only
// once. The queue is reported after the debounce time passed without
// further events.
//
// The watcher can be paused, while the app changes the files of the package
// itself (e.g. while the thumbnails are generated). The events that arrive
// while the watcher is paused are still queued, so that no change is lost.
// They are reported when the watcher is resumed.
type packageWatcher struct {
	// debounce is the time to wait for further events, before the queued
	// paths are reported.
	debounce time.Duration

	// onUpdate is called with the paths that changed. The full flag is set
	// when the events of the watcher were lost (e.g. because the buffer of
	// the operating system overflowed), so that the package is scanned
	// completely, instead of updating only the reported paths.
	onUpdate func(paths []string, full bool)

	// onError is called for the errors the watcher runs into.
	onError func(err error)

	watcher *fsnotify.Watcher

	mu sync.Mutex
	// queue holds the paths that changed and are not reported yet.
	queue *structures.Set[string]
	// timer reports the queued paths after the debounce time.
	timer *time.Timer
	// generation counts the started timers, so that a timer that was
	// stopped does not report the queued paths anymore.
	generation uint64
	// paused is set while the queued paths are not reported.
	paused bool
	// closed is set when the watcher was closed.
	closed bool
	// overflowed is set when the events of the watcher were lost.
	overflowed bool
}

// newPackageWatcher creates a watcher, that reports the changed paths to
// onUpdate and the errors it runs into to onError.
func newPackageWatcher(
	debounce time.Duration,
	onUpdate func(paths []string, full bool),
	onError func(err error),
) (*packageWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	if debounce <= 0 {
		debounce = defaultPackageWatcherDebounce
	}

	pw := &packageWatcher{
		debounce: debounce,
		onUpdate: onUpdate,
		onError:  onError,
		watcher:  watcher,
		queue:    structures.NewSet[string](),
	}

	go pw.watch()

	return pw, nil
}

// watch reads the events of the watcher, until it is closed.
func (pw *packageWatcher) watch() {
	for {
		select {
		case event, ok := <-pw.watcher.Events:
			if !ok {
				return
			}
			// the mode of a file is not a change of its content, but the
			// watcher reports it for every file it sees
			if event.Has(fsnotify.Chmod) {
				continue
			}
			pw.enqueue(event.Name)
		case err, ok := <-pw.watcher.Errors:
			if !ok {
				return
			}
			if errors.Is(err, fsnotify.ErrEventOverflow) {
				// The events of the watcher were lost, so the package has to
				// be scanned completely, to find all the changes that were
				// not reported.
				pw.overflow()
			}
			if pw.onError != nil {
				pw.onError(err)
			}
		}
	}
}

// enqueue records a changed path and (re)starts the debounce timer.
func (pw *packageWatcher) enqueue(path string) {
	pw.mu.Lock()
	if pw.closed {
		pw.mu.Unlock()
		return
	}
	pw.queue.Add(path)
	pw.armTimerLocked()
	pw.mu.Unlock()

	// a new folder has to be watched, so that the changes of the files in
	// it are noticed
	pw.watchNewFolder(path)
}

// overflow records that the events of the watcher were lost.
func (pw *packageWatcher) overflow() {
	pw.mu.Lock()
	defer pw.mu.Unlock()
	if pw.closed {
		return
	}
	pw.overflowed = true
	pw.armTimerLocked()
}

// armTimerLocked (re)starts the debounce timer.
//
// The caller has to hold the lock of the watcher.
func (pw *packageWatcher) armTimerLocked() {
	pw.stopTimerLocked()

	if pw.paused || pw.closed {
		return
	}
	if pw.queue.Size() == 0 && !pw.overflowed {
		return
	}

	pw.generation++
	generation := pw.generation
	pw.timer = time.AfterFunc(pw.debounce, func() {
		pw.report(generation)
	})
}

// stopTimerLocked stops the debounce timer.
//
// The caller has to hold the lock of the watcher.
func (pw *packageWatcher) stopTimerLocked() {
	if pw.timer != nil {
		pw.timer.Stop()
		pw.timer = nil
	}
	// the timers that were started before are not allowed to report the
	// queued paths anymore
	pw.generation++
}

// report hands the queued paths to the update callback of the watcher.
func (pw *packageWatcher) report(generation uint64) {
	pw.mu.Lock()
	if pw.closed || pw.paused || generation != pw.generation {
		pw.mu.Unlock()
		return
	}
	paths := pw.queue.AsSlice()
	full := pw.overflowed
	pw.queue = structures.NewSet[string]()
	pw.overflowed = false
	pw.mu.Unlock()

	pw.cleanupWatchList(paths)

	if len(paths) == 0 && !full {
		return
	}
	if pw.onUpdate != nil {
		pw.onUpdate(paths, full)
	}
}

// cleanupWatchList removes the folders that do not exist anymore from the
// watch list.
func (pw *packageWatcher) cleanupWatchList(paths []string) {
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			continue
		}
		if !slices.Contains(pw.watcher.WatchList(), path) {
			continue
		}
		if err := pw.watcher.Remove(path); err != nil {
			log.WithError(err).
				WithField("path", path).
				Warn("could not remove the deleted folder from the watch list")
		}
	}
}

// watchNewFolder adds the given path to the watch list, if it is a folder.
//
// The subfolders of a new folder are watched too, because they can be
// created before the folder itself is watched.
func (pw *packageWatcher) watchNewFolder(path string) {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return
	}
	if slices.Contains(pw.watcher.WatchList(), path) {
		return
	}
	if err := pw.watcher.Add(path); err != nil {
		log.WithError(err).
			WithField("path", path).
			Warn("could not watch the new folder")
		return
	}

	_, dirs, _ := utils.ListDir(path)
	for _, sub := range dirs {
		if slices.Contains(pw.watcher.WatchList(), sub) {
			continue
		}
		if err := pw.watcher.Add(sub); err != nil {
			log.WithError(err).
				WithField("path", sub).
				Warn("could not watch the new folder")
		}
	}
}

// Watch adds the given folder and all of its subfolders to the watch list.
func (pw *packageWatcher) Watch(dir string) error {
	if dir == "" {
		return nil
	}

	if err := pw.watcher.Add(dir); err != nil {
		return err
	}

	_, dirs, errs := utils.ListDir(dir)
	for _, err := range errs {
		log.WithError(err).
			WithField("dir", dir).
			Warn("could not list the folder to watch")
	}
	for _, sub := range dirs {
		if err := pw.watcher.Add(sub); err != nil {
			log.WithError(err).
				WithField("path", sub).
				Warn("could not watch the folder")
		}
	}

	return nil
}

// Pause stops the reporting of the queued paths.
//
// The events that arrive while the watcher is paused are still queued, so
// that no change is lost.
func (pw *packageWatcher) Pause() {
	pw.mu.Lock()
	defer pw.mu.Unlock()
	pw.paused = true
	pw.stopTimerLocked()
}

// Resume restarts the reporting of the queued paths.
func (pw *packageWatcher) Resume() {
	pw.mu.Lock()
	defer pw.mu.Unlock()
	pw.paused = false
	pw.armTimerLocked()
}

// DiscardUnder forgets the queued changes of the files under the given
// folder.
//
// It is used by the tasks that change the files of the package themselves
// and update the affected parts of the package themselves, so that their
// changes are not reported a second time.
//
// An empty folder discards all the queued changes.
func (pw *packageWatcher) DiscardUnder(dir string) {
	pw.mu.Lock()
	defer pw.mu.Unlock()

	if dir == "" {
		pw.queue = structures.NewSet[string]()
		return
	}

	prefix := filepath.Clean(dir) + string(filepath.Separator)
	for _, path := range pw.queue.AsSlice() {
		if strings.HasPrefix(path, prefix) {
			pw.queue.Remove(path)
		}
	}
}

// Pending returns the paths that changed and are not reported yet.
func (pw *packageWatcher) Pending() []string {
	pw.mu.Lock()
	defer pw.mu.Unlock()
	return pw.queue.AsSlice()
}

// Close stops the watcher and releases its resources.
func (pw *packageWatcher) Close() {
	pw.mu.Lock()
	if pw.closed {
		pw.mu.Unlock()
		return
	}
	pw.closed = true
	pw.stopTimerLocked()
	pw.mu.Unlock()

	if err := pw.watcher.Close(); err != nil {
		log.WithError(err).Warn("could not close the filesystem watcher")
	}
}
