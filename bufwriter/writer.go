// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package bufwriter

import (
	"context"
	"errors"
	"io"
	"sync"
)

// ErrClosed reports a write after Close starts.
var ErrClosed = errors.New("bufwriter: closed")

// ErrPanicked reports a writer panic without disclosing its payload.
var ErrPanicked = errors.New("bufwriter: writer panicked")

type entry struct {
	bytes   []byte
	barrier chan struct{}
}

// Writer queues owned byte copies. Overflow writes synchronously and may
// overtake queued writes. Close does not close the caller's underlying writer.
type Writer struct {
	mu      sync.Mutex
	writeMu chan struct{}
	errMu   sync.Mutex
	sink    io.Writer
	queue   chan entry
	done    chan struct{}
	stop    chan struct{}
	closed  bool
	err     error
	onError func(error)
	pending map[chan struct{}]struct{}
}

// New validates the queue size and starts one worker.
// onError must be concurrency-safe and must not reenter Writer.
// Callback panics are contained.
func New(sink io.Writer, opts ...Option) (*Writer, error) {
	o := newOptions(opts...)
	if sink == nil || o.size <= 0 {
		return nil, errors.New("bufwriter: writer and positive size required")
	}
	w := &Writer{sink: sink, writeMu: make(chan struct{}, 1), queue: make(chan entry, o.size),
		done: make(chan struct{}), stop: make(chan struct{}), onError: o.onError, pending: make(map[chan struct{}]struct{})}
	go w.run()
	return w, nil
}
func (w *Writer) run() {
	defer close(w.done)
	defer func() {
		w.mu.Lock()
		pending := w.pendingWrites()
		w.mu.Unlock()
		for _, done := range pending {
			<-done
		}
	}()
	for {
		select {
		case e := <-w.queue:
			w.process(e)
		case <-w.stop:
			for {
				select {
				case e := <-w.queue:
					w.process(e)
				default:
					return
				}
			}
		}
	}
}
func (w *Writer) process(e entry) {
	if e.barrier != nil {
		close(e.barrier)
	} else {
		_, _ = w.write(e.bytes)
	}
}

func (w *Writer) write(p []byte) (n int, err error) {
	func() {
		w.writeMu <- struct{}{}
		defer func() { <-w.writeMu }()
		defer func() {
			if recover() != nil {
				n = 0
				err = ErrPanicked
			}
		}()
		n, err = w.sink.Write(p)
		if n != len(p) && err == nil {
			err = io.ErrShortWrite
		}
	}()
	if err != nil {
		w.errMu.Lock()
		w.err = errors.Join(w.err, err)
		w.errMu.Unlock()
		if w.onError != nil {
			func() { defer func() { _ = recover() }(); w.onError(err) }()
		}
	}
	return n, err
}

// Write copies bytes before enqueueing. A full queue falls back to synchronous I/O.
func (w *Writer) Write(p []byte) (int, error) {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		if w.onError != nil {
			func() { defer func() { _ = recover() }(); w.onError(ErrClosed) }()
		}
		return w.write(p)
	}
	owned := append([]byte(nil), p...)
	select {
	case w.queue <- entry{bytes: owned}:
		w.mu.Unlock()
		return len(p), nil
	default:
		done := make(chan struct{})
		w.pending[done] = struct{}{}
		w.mu.Unlock()
		n, err := w.write(owned)
		w.mu.Lock()
		delete(w.pending, done)
		close(done)
		w.mu.Unlock()
		return n, err
	}
}
func (w *Writer) pendingWrites() []chan struct{} {
	pending := make([]chan struct{}, 0, len(w.pending))
	for done := range w.pending {
		pending = append(pending, done)
	}
	return pending
}
func (w *Writer) errors() error {
	w.errMu.Lock()
	defer w.errMu.Unlock()
	err := w.err
	w.err = nil
	return err
}

// Flush waits for previously admitted writes or returns the context error.
// It returns and resets joined write errors since the previous Flush or Close.
// onError receives every individual error.
// After Close finishes it returns immediately, including after write panics.
func (w *Writer) Flush(ctx context.Context) error {
	barrier := make(chan struct{})
	w.mu.Lock()
	pending := w.pendingWrites()
	if w.closed {
		w.mu.Unlock()
		select {
		case <-w.done:
			return w.errors()
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	w.mu.Unlock()
	select {
	case w.queue <- entry{barrier: barrier}:
	case <-ctx.Done():
		return ctx.Err()
	case <-w.done:
		return w.errors()
	}

	select {
	case <-barrier:
		for _, done := range pending {
			select {
			case <-done:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return w.errors()
	case <-w.done:
		return w.errors()
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Close starts draining exactly once. A timed-out call does not abandon draining;
// a later call can wait for completion. The underlying writer remains open.
func (w *Writer) Close(ctx context.Context) error {
	w.mu.Lock()
	if !w.closed {
		w.closed = true
		close(w.stop)
	}
	w.mu.Unlock()
	select {
	case <-w.done:
		return w.errors()
	case <-ctx.Done():
		return ctx.Err()
	}
}
