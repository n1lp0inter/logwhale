package logwhale

import (
	"context"
	"fmt"
	"path"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

var (
	logBufferLen = 1024 // default buffer length for the log reader
)

// Option is a function that can be passed to NewLogManager to configure it.
type Option func(*LogManager) error

type logWatcher struct {
	logfile *logFile
	cancel  context.CancelFunc
}

// LogManager is used to watch one of more log files for changes and consume the data line by line to be passed
// as a byte slice to a consumer channel.
type LogManager struct {
	closed    bool // closed is a flag that indicates if the LogManager has been closed
	ctx       context.Context
	ctxCancel context.CancelFunc

	evwCancelChan chan error // event watcher cancel channel
	fileWatcher   *fsnotify.Watcher

	lwMutex     *sync.RWMutex          // lwMutex is a mutex for the logsWatched map
	logsWatched map[string]*logWatcher // logsWatched is a map of log files being watched

	pwMutex      *sync.RWMutex  // pwMutex is a mutex for the pathsWatched map
	pathsWatched map[string]int // pathsWatched is a map of paths being watched

	removeChan chan string // removeChan is a channel for log files to be removed from the logsWatched map
}

// NewLogManager creates a new LogManager
func NewLogManager(ctx context.Context, options ...Option) (*LogManager, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("unable to create file watcher: %w", err)
	}

	// Context for LogManager cancellation
	if ctx == nil {
		return nil, fmt.Errorf("LogManager requires a valid context")
	}
	ctx, cancel := context.WithCancel(ctx)

	lm := &LogManager{
		ctx:          ctx,
		ctxCancel:    cancel,
		fileWatcher:  watcher,
		logsWatched:  make(map[string]*logWatcher),
		lwMutex:      &sync.RWMutex{},
		pathsWatched: make(map[string]int),
		pwMutex:      &sync.RWMutex{},

		evwCancelChan: make(chan error),
		removeChan:    make(chan string),
	}

	if err := lm.withOptions(options...); err != nil {
		return nil, fmt.Errorf("unable to process options: %w", err)
	}

	// Start Processing Events
	lm.eventWatcher()

	// Start Listening for removals
	lm.logfileRemover()

	return lm, nil
}

// AddLogFile adds a log file to the LogManager and starts its data processor.
func (lm *LogManager) AddLogFile(lp string) (<-chan []byte, <-chan error, error) {
	if lm.closed {
		return nil, nil, fmt.Errorf("log manager closed")
	}

	// Clean up the file path
	lfp := path.Clean(lp)
	lf, err := newLogFile(lfp)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to configure log file: %w", err)
	}

	lm.lwMutex.Lock()
	defer lm.lwMutex.Unlock()
	if _, exists := lm.logsWatched[lfp]; exists {
		return nil, nil, fmt.Errorf("log file already being watched: %s", lfp)
	}

	// Create a context for the log file
	ctx, cancel := context.WithCancel(lm.ctx)
	lm.logsWatched[lfp] = &logWatcher{logfile: lf, cancel: cancel}

	lm.pwMutex.Lock()
	defer lm.pwMutex.Unlock()
	if _, exists := lm.pathsWatched[lf.basepath]; exists {
		lm.pathsWatched[lf.basepath]++
	} else {
		err := lm.fileWatcher.Add(lf.basepath)
		if err != nil {
			delete(lm.logsWatched, lfp) // Remove the log file from the logsWatched map
			return nil, nil, fmt.Errorf("unable to watch base path: %w", err)
		}
		lm.pathsWatched[lf.basepath] = 1
	}

	// Start processing
	lf.dataProcessor(ctx, lm.evwCancelChan, lm.removeChan)

	return lf.dataChan, lf.errorChan, nil
}

// RemoveLogFile removes a log file from the LogManager and stops its data processor.
func (lm *LogManager) RemoveLogFile(lp string) error {
	if lm.closed {
		return fmt.Errorf("log manager closed")
	}

	// Clean up the file path
	lfp := path.Clean(lp)
	bp := path.Dir(lfp)

	lm.lwMutex.Lock()
	defer lm.lwMutex.Unlock()
	lw, exists := lm.logsWatched[lfp]
	if !exists {
		return fmt.Errorf("log file (%s) not watched", lfp)
	}
	delete(lm.logsWatched, lfp)
	lw.cancel()

	lm.pwMutex.Lock()
	defer lm.pwMutex.Unlock()
	if _, exists := lm.pathsWatched[bp]; !exists {
		return fmt.Errorf("base path (%s) not watched", bp)
	}

	if lm.pathsWatched[bp] == 1 {
		delete(lm.pathsWatched, bp)
		err := lm.fileWatcher.Remove(bp)
		if err != nil {
			return fmt.Errorf("unable to unwatch base path (%s): %w", bp, err)
		}
	} else {
		lm.pathsWatched[bp]--
	}

	return nil
}

func (lm *LogManager) logfileRemover() {
	go func() {
		for {
			select {
			case <-lm.ctx.Done():
				return
			case fp := <-lm.removeChan:
				lm.RemoveLogFile(fp)
			}
		}
	}()
}

// eventWatcher is a goroutine that watches for file system events and sends them to the appropriate log file.
func (lm *LogManager) eventWatcher() {
	go func() {
		for {
			select {
			case <-lm.ctx.Done():
				return
			case fse, ok := <-lm.fileWatcher.Events:
				if !ok {
					lm.evwCancelChan <- fmt.Errorf("file watcher closed unexpectedly")
					return
				}

				lm.lwMutex.RLock()
				lw, exists := lm.logsWatched[fse.Name]
				if !exists {
					lm.lwMutex.RUnlock()
					continue
				}

				if fse.Op.Has(fsnotify.Create) {
					select {
					case lw.logfile.stateEvents <- stateEventCreated:
					default:
					}
					lm.lwMutex.RUnlock()
					continue
				}

				if fse.Op.Has(fsnotify.Remove) {
					select {
					case lw.logfile.stateEvents <- stateEventRemoved:
					default:
					}
					lm.lwMutex.RUnlock()
					continue
				}

				if fse.Op.Has(fsnotify.Write) {
					// pop off the lastWriteEvent channel
					select {
					case <-lw.logfile.lastWriteEvent:
					default:
					}

					// Write a new value to the lastWriteEvent channel
					select {
					case lw.logfile.lastWriteEvent <- time.Now():
					default:
					}
					lm.lwMutex.RUnlock()
					continue
				}
			case err := <-lm.fileWatcher.Errors:
				lm.evwCancelChan <- fmt.Errorf("file watcher error: %w", err)
				return
			}
		}
	}()
}

// Close closes the LogManager.
func (lm *LogManager) Close() error {
	lm.ctxCancel()
	lm.closed = true

	lm.lwMutex.Lock()
	defer lm.lwMutex.Unlock()
	lm.logsWatched = nil

	lm.pwMutex.Lock()
	defer lm.pwMutex.Unlock()
	lm.pathsWatched = nil

	if err := lm.fileWatcher.Close(); err != nil {
		return fmt.Errorf("could not close file watcher: %w", err)
	}

	return nil
}

// WithOptions applies the given options to the LogManager
func (lm *LogManager) withOptions(opts ...Option) error {
	for _, opt := range opts {
		err := opt(lm)
		if err != nil {
			return fmt.Errorf("cannot apply Option: %w", err)
		}
	}
	return nil
}
