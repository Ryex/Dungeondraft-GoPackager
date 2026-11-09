package gui

import (
	"errors"
	"os"
	"path/filepath"
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

	// root is the folder of the package that is watched. The symlinks
	// that leave it are not followed, so that the files outside of the
	// package are not watched through them.
	root string
	// watched caches the paths that were added to the watcher, so that
	// the watch list of the watcher does not have to be read to know
	// whether a path is watched already.
	watched map[string]bool
	// watchMu guards root and watched.
	watchMu sync.Mutex

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
		watched:  make(map[string]bool),
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
		if _, err := os.Lstat(path); err == nil {
			continue
		}
		clean := filepath.Clean(path)
		pw.watchMu.Lock()
		watched := pw.watched[clean]
		if watched {
			delete(pw.watched, clean)
		}
		pw.watchMu.Unlock()
		if !watched {
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
//
// Symlinks are only followed when they stay inside the root of the
// package, so that the files outside of the package are not watched
// through them.
func (pw *packageWatcher) watchNewFolder(path string) {
	info, err := os.Lstat(path)
	if err != nil {
		return
	}

	if info.Mode()&os.ModeSymlink != 0 {
		pw.watchMu.Lock()
		root := pw.root
		pw.watchMu.Unlock()
		if !symlinkInRoot(root, path) {
			return
		}
		// the link stays inside the package, so the folder it points to
		// is watched through it
		info, err = os.Stat(path)
		if err != nil {
			return
		}
	}
	if !info.IsDir() {
		return
	}

	clean := filepath.Clean(path)
	pw.watchMu.Lock()
	alreadyWatched := pw.watched[clean]
	pw.watchMu.Unlock()
	if alreadyWatched {
		return
	}

	_, dirs, _ := utils.ListDir(path)

	pw.watchMu.Lock()
	defer pw.watchMu.Unlock()

	if pw.watched[clean] {
		return
	}
	if err := pw.watcher.Add(path); err != nil {
		log.WithError(err).
			WithField("path", path).
			Warn("could not watch the new folder")
		return
	}
	pw.watched[clean] = true

	for _, sub := range dirs {
		// symlinks that leave the root of the package are not followed
		if subInfo, err := os.Lstat(sub); err == nil &&
			subInfo.Mode()&os.ModeSymlink != 0 && !symlinkInRoot(pw.root, sub) {
			continue
		}
		if pw.watched[filepath.Clean(sub)] {
			continue
		}
		if err := pw.watcher.Add(sub); err != nil {
			log.WithError(err).
				WithField("path", sub).
				Warn("could not watch the new folder")
			continue
		}
		pw.watched[filepath.Clean(sub)] = true
	}
}

// symlinkInRoot reports whether the symlink at the given path resolves to
// a target inside the root of the package.
//
// The caller has to make sure that the path is a symlink.
func symlinkInRoot(root string, path string) bool {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return false
	}
	isSub, _ := utils.PathIsSub(root, resolved)
	return isSub
}

// isWatched reports whether the given path is in the watch list.
func (pw *packageWatcher) isWatched(path string) bool {
	pw.watchMu.Lock()
	defer pw.watchMu.Unlock()
	return pw.watched[filepath.Clean(path)]
}

// Watch adds the given folder and all of its subfolders to the watch list.
//
// The root of the package is remembered, so that the symlinks that leave
// it are not followed when new folders are watched.
func (pw *packageWatcher) Watch(dir string) error {
	if dir == "" {
		return nil
	}

	if err := pw.watcher.Add(dir); err != nil {
		return err
	}

	pw.watchMu.Lock()
	pw.root = filepath.Clean(dir)
	pw.watched[filepath.Clean(dir)] = true
	pw.watchMu.Unlock()

	_, dirs, errs := utils.ListDir(dir)
	for _, err := range errs {
		log.WithError(err).
			WithField("dir", dir).
			Warn("could not list the folder to watch")
	}
	pw.watchMu.Lock()
	defer pw.watchMu.Unlock()
	for _, sub := range dirs {
		// symlinks that leave the root of the package are not followed
		if subInfo, err := os.Lstat(sub); err == nil &&
			subInfo.Mode()&os.ModeSymlink != 0 && !symlinkInRoot(pw.root, sub) {
			continue
		}
		if pw.watched[filepath.Clean(sub)] {
			continue
		}
		if err := pw.watcher.Add(sub); err != nil {
			log.WithError(err).
				WithField("path", sub).
				Warn("could not watch the folder")
			continue
		}
		pw.watched[filepath.Clean(sub)] = true
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
//
// The given path itself counts as under itself, so that a single file can
// be discarded too.
func (pw *packageWatcher) DiscardUnder(dir string) {
	pw.mu.Lock()
	defer pw.mu.Unlock()

	if dir == "" {
		pw.queue = structures.NewSet[string]()
		return
	}

	clean := filepath.Clean(dir)
	prefix := clean + string(filepath.Separator)
	for _, path := range pw.queue.AsSlice() {
		if path == clean || strings.HasPrefix(path, prefix) {
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
