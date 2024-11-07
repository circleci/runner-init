package step

import (
	"bufio"
	"errors"
	"io"
	"sync"
	"time"
)

var ErrWriterClosed = errors.New("BufferingWriter was closed")

// BufferingWriter is a writer that wraps another io.Writer and
// buffers calls to Write().  The buffer is flushed every `flushTimeout` or
// if the buffer size exceeds `flushBufferSize` after a write.  It will
// never write more than `flushBufferSize` bytes at once to the underlying
// writer.
type BufferingWriter struct {
	w            *bufio.Writer
	flushTimeout time.Duration
	notifyErrors chan error

	data chan []byte

	shutdown     chan chan struct{}
	shutdownLock sync.RWMutex
}

// NewBufferingWriter constructs a BufferingWriter.  If notifyErrors is provided,
// then any errors writing to the underlying writer will be sent to that channel.
// If notifyErrors is nil, errors will be discarded.
func NewBufferingWriter(w io.Writer, flushTimeout time.Duration, flushBufferSize int,
	notifyErrors chan error) *BufferingWriter {
	bufWriter := bufio.NewWriterSize(w, flushBufferSize)
	bw := &BufferingWriter{
		w:            bufWriter,
		flushTimeout: flushTimeout,
		notifyErrors: notifyErrors,
		data:         make(chan []byte),
		shutdown:     make(chan chan struct{}),
	}

	go bw.run()
	return bw
}

func (bw *BufferingWriter) notifyIfError(err error) {
	if err != nil && bw.notifyErrors != nil {
		select {
		case bw.notifyErrors <- err:
		default:
		}
	}
}

// run will flush data when the timeout expires, or when the buffer exceeds
// the given size
func (bw *BufferingWriter) run() {
	ticker := time.NewTicker(bw.flushTimeout)
	defer ticker.Stop()

	for {
		select {
		case data, ok := <-bw.data:
			if !ok {
				return // this shouldn't ever happen
			}

			_, err := bw.w.Write(data)
			bw.notifyIfError(err)

		case <-ticker.C:
			bw.notifyIfError(bw.w.Flush())

		case replyCh := <-bw.shutdown:
			bw.notifyIfError(bw.w.Flush())
			replyCh <- struct{}{}
			return
		}
	}
}

// Write will enqueue the data written.  The only error that can be
// returned is iohelper.ErrWriterClosed if Write() is called after
// Close().  Otherwise it always returns no error.
func (bw *BufferingWriter) Write(p []byte) (n int, err error) {
	bw.shutdownLock.RLock()
	defer bw.shutdownLock.RUnlock()

	if bw.shutdown == nil {
		return 0, ErrWriterClosed
	}

	data := make([]byte, len(p))
	copy(data, p)
	bw.data <- data

	return len(p), nil
}

func (bw *BufferingWriter) WriteString(s string) (n int, err error) {
	bw.shutdownLock.RLock()
	defer bw.shutdownLock.RUnlock()

	if bw.shutdown == nil {
		return 0, ErrWriterClosed
	}

	p := []byte(s)
	bw.data <- p

	return len(p), nil
}

// Close always returns nil to implement io.Closer
func (bw *BufferingWriter) Close() error {
	bw.shutdownLock.Lock()
	defer bw.shutdownLock.Unlock()

	if bw.shutdown != nil {
		replyCh := make(chan struct{})
		bw.shutdown <- replyCh
		<-replyCh
		bw.shutdown = nil
	}

	return nil
}
