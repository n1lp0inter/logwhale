package logwhale

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"time"
)

type stateEventOp uint8

func (so stateEventOp) String() string {
	switch so {
	case stateEventCreated:
		return "created"
	case stateEventRemoved:
		return "removed"
	default:
		return "unknown"
	}
}

const (
	stateEventCreated stateEventOp = iota
	stateEventRemoved
)

// logFile is a struct that represents a log file to be consumed by the log manager
type logFile struct {
	filePath       string
	basepath       string
	created        bool
	lastRead       time.Time
	dataChan       chan []byte
	errorChan      chan error
	lastWriteEvent chan time.Time
	stateEvents    chan stateEventOp
}

// newLogFile configures the logFile struct
func newLogFile(fp string, bs int) (*logFile, error) {
	fi, _ := os.Stat(fp)
	if fi != nil && fi.IsDir() {
		return nil, NewLogWhaleError(ErrorStateFilePath, fmt.Sprintf("filepath (%s) is a directory, must be a file", fp), nil)
	}

	lf := logFile{
		filePath:       fp,
		basepath:       path.Dir(fp),
		created:        true,
		dataChan:       make(chan []byte, bs),
		errorChan:      make(chan error, 1),
		lastWriteEvent: make(chan time.Time, 1),
		stateEvents:    make(chan stateEventOp),
	}

	return &lf, nil
}

// dataProcessor is a goroutine that processes the log file data.
//
// ctx is the context for the log manager that can be used to cancel the entire process
// ewCancelChan is a channel used in cases of event watcher cancellation to cancel the data processors for all watched files
// stopChan is a channel used to signal the log manager that the data processor has stopped
//
// The data processor will read the log file line by line and send the data to the data channel.
// If the reader encounters an EOF, it will wait for a write event that has occurred after the most recent read operation before continuing.
// If the reader encounters more critical errors, it will send the error to the error channel and stop the data processor.
// If the file does not exist at startup, the data processor will wait for the file to be created before continuing.
//
// The data processor expects the log data to be newline delimited
func (lf *logFile) dataProcessor(ctx context.Context, ewCancelChan <-chan error, stopChan chan<- string) {
	go func() {
		defer close(lf.dataChan)
		defer close(lf.errorChan)
		defer close(lf.lastWriteEvent)
		defer close(lf.stateEvents)

	creationLoop:
		for {
			// Stat the file to see if it exists and is a file
			fi, err := os.Stat(lf.filePath)
			if err != nil {
				if !os.IsNotExist(err) {
					stopChan <- lf.filePath
					lf.errorChan <- NewLogWhaleError(ErrorStateInternal, fmt.Sprintf("unhandlable error encountered with path (%s)", lf.filePath), err)
					return
				}
				lf.created = false
			}

			if fi != nil && fi.IsDir() {
				stopChan <- lf.filePath
				lf.errorChan <- NewLogWhaleError(ErrorStateFilePath, fmt.Sprintf("filepath (%s) is a directory, must be a file", lf.filePath), nil)
				return
			}

			// If the file doesn't exist, wait for it to be created
			if !lf.created {
				select {
				case <-ctx.Done():
					return
				case so := <-lf.stateEvents:
					if so == stateEventCreated {
						lf.created = true
						break
					}
					stopChan <- lf.filePath
					lf.errorChan <- NewLogWhaleError(ErrorStateInternal, fmt.Sprintf("unexpected state event while waiting for file creation: %s", so), nil)
					return
				}
			}

			// Open the file once created
			of, err := os.Open(lf.filePath)
			if err != nil {
				stopChan <- lf.filePath
				lf.errorChan <- NewLogWhaleError(ErrorStateFileIO, fmt.Sprintf("unable to open file: %s", lf.filePath), err)
				return
			}
			defer of.Close()

			// Main file read loop
			fr := bufio.NewReader(of)
		readLoop:
			for {
				var readErr error
				for {
					var bl []byte
					bl, readErr = fr.ReadBytes('\n') // Read lines, but we'll still collect the bytes without the newline character

					bl = bytes.Trim(bl, "\n") // Trim the newline character from the end of the line if it exists

					// Skip empty data
					if len(bl) == 0 {
						if readErr != nil {
							break
						}
						continue
					}

					// Send the line to the data channel
					select {
					case <-ctx.Done():
						return
					case lf.dataChan <- bl:
					}

					// If there is an error state then break the loop before next read
					if readErr != nil {
						break
					}
				}

				if readErr == io.EOF {
					lf.lastRead = time.Now()
				} else {
					stopChan <- lf.filePath
					lf.errorChan <- NewLogWhaleError(ErrorStateFileIO, fmt.Sprintf("error reading file: %s", lf.filePath), readErr)
					return
				}

				for {
					select {
					case <-ctx.Done():
						return
					case we := <-lf.lastWriteEvent:
						if we.After(lf.lastRead) {
							continue readLoop
						}
						continue
					case err := <-ewCancelChan:
						stopChan <- lf.filePath
						lf.errorChan <- NewLogWhaleError(ErrorStateCancelled, fmt.Sprintf("operation cancelled"), err)
						return
					case so := <-lf.stateEvents:
						if so == stateEventRemoved {
							lf.errorChan <- NewLogWhaleError(ErrorStateFileRemoved, fmt.Sprintf("file removed, waiting for creation: %s", lf.filePath), nil)
							lf.created = false
							of.Close()
							continue creationLoop
						}
						stopChan <- lf.filePath
						lf.errorChan <- NewLogWhaleError(ErrorStateInternal, fmt.Sprintf("unexpected state event waiting for writing to resume: %s", so), nil)
						continue
					}
				}
			}
		}
	}()
}
