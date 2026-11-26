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

// ErrDropped reports a write rejected by the queue's admission policy.
var ErrDropped = errors.New("bufwriter: write dropped")

// Stats counts writes, including any that are in flight when Close returns.
type Stats struct {
	Written         uint64
	Failed          uint64
	Pending         uint64
	Dropped         uint64 // Includes DroppedPriority.
	DroppedPriority uint64
}

type entry struct {
	bytes  []byte
	result chan error
}

// Writer queues owned byte copies. Only its worker writes to the sink.
type Writer struct {
	mu      sync.Mutex
	sink    io.Writer
	queue   chan entry
	done    chan struct{}
	changed chan struct{}
	closed  bool
	waiting int
	stats   Stats
	err     error
	onError func(error)
}

// New validates the queue size and starts one worker.
// onError receives sink failures on the worker and must return promptly.
// Callback panics are contained.
func New(sink io.Writer, opts ...Option) (*Writer, error) {
	o := newOptions(opts...)
	if sink == nil || o.size <= 0 {
		return nil, errors.New("bufwriter: writer and positive size required")
	}

	w := &Writer{sink: sink, queue: make(chan entry, o.size), done: make(chan struct{}), onError: o.onError}
	go w.run()

	return w, nil
}

func (w *Writer) run() {
	defer close(w.done)

	for e := range w.queue {
		w.mu.Lock()
		w.notify()
		w.mu.Unlock()

		err := w.write(e.bytes)

		w.mu.Lock()

		w.stats.Pending--
		if err == nil {
			w.stats.Written++
		} else {
			w.stats.Failed++
			w.err = errors.Join(w.err, err)
		}

		w.notify()
		w.mu.Unlock()

		if e.result != nil {
			e.result <- err
		}
	}
}

func (w *Writer) write(p []byte) (err error) {
	defer func() {
		if recover() != nil {
			err = ErrPanicked
		}

		if err != nil && w.onError != nil {
			func() { defer func() { _ = recover() }(); w.onError(err) }()
		}
	}()

	n, err := w.sink.Write(p)
	if n != len(p) && err == nil {
		err = io.ErrShortWrite
	}

	return err
}

// notify wakes all waiters. The caller holds mu.
func (w *Writer) notify() {
	if w.changed != nil {
		close(w.changed)
		w.changed = nil
	}
}

// changedSignal returns a signal for the next state change. The caller holds mu.
func (w *Writer) changedSignal() <-chan struct{} {
	if w.changed == nil {
		w.changed = make(chan struct{})
	}

	return w.changed
}

// Write copies and queues p, or returns ErrDropped immediately if capacity is
// exhausted or reserved for waiting priority writes.
func (w *Writer) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return 0, ErrClosed
	}

	if len(w.queue) == cap(w.queue) || w.waiting != 0 {
		w.stats.Dropped++
		return 0, ErrDropped
	}

	w.queue <- entry{bytes: append([]byte(nil), p...)}

	w.stats.Pending++

	return len(p), nil
}

// WritePriority waits for queue space until ctx ends. An admission timeout
// drops the write. An admitted write stays queued even if ctx ends while
// waiting for its delivery result.
func (w *Writer) WritePriority(ctx context.Context, p []byte) error {
	result, err := w.enqueuePriority(ctx, p)
	if err != nil {
		return err
	}

	select {
	case err := <-result:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// DropPriority records a priority write whose preparation exhausted its budget.
func (w *Writer) DropPriority(err error) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return ErrClosed
	}

	return w.dropPriority(err)
}

func (w *Writer) dropPriority(err error) error {
	w.stats.Dropped++
	w.stats.DroppedPriority++

	return errors.Join(ErrDropped, err)
}

func (w *Writer) enqueuePriority(ctx context.Context, p []byte) (<-chan error, error) {
	w.mu.Lock()

	w.waiting++
	defer func() {
		w.waiting--
		w.mu.Unlock()
	}()

	for {
		if w.closed {
			return nil, ErrClosed
		}

		if err := ctx.Err(); err != nil {
			return nil, w.dropPriority(err)
		}

		if len(w.queue) < cap(w.queue) {
			result := make(chan error, 1)
			w.queue <- entry{bytes: append([]byte(nil), p...), result: result}

			w.stats.Pending++

			return result, nil
		}

		changed := w.changedSignal()
		w.mu.Unlock()

		select {
		case <-changed:
		case <-ctx.Done():
		}

		w.mu.Lock()
	}
}

// Flush waits for previously queued writes or returns the context error.
// A completed Flush returns and resets delivery errors since the last Flush.
func (w *Writer) Flush(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	target := w.stats.Written + w.stats.Failed + w.stats.Pending
	for w.stats.Written+w.stats.Failed < target {
		changed := w.changedSignal()
		w.mu.Unlock()

		select {
		case <-changed:
		case <-ctx.Done():
			w.mu.Lock()
			return ctx.Err()
		}

		w.mu.Lock()
	}

	err := w.err
	w.err = nil

	return err
}

// Close stops admission and waits for the worker until ctx ends. It returns a
// snapshot of delivery counts and any sink errors since the last Flush. If
// ctx ends first, the worker keeps draining and the caller can retry Close.
// The caller retains ownership of the sink.
func (w *Writer) Close(ctx context.Context) (Stats, error) {
	w.mu.Lock()
	if !w.closed {
		w.closed = true
		close(w.queue)
		w.notify()
	}
	w.mu.Unlock()

	var err error

	select {
	case <-w.done:
	case <-ctx.Done():
		select {
		case <-w.done:
		default:
			err = ctx.Err()
		}
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	return w.stats, errors.Join(w.err, err)
}
