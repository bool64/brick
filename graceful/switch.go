package graceful

import (
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
)

// ErrTimeout describes tasks that failed to finish in time.
type ErrTimeout []string

// Error returns an error message.
func (e ErrTimeout) Error() string {
	return "shutdown timeout, tasks left: " + strings.Join(e, ", ")
}

// Switch is graceful shutdown handler.
//
// Please use NewSwitch to create an instance.
type Switch struct {
	sig chan os.Signal

	done <-chan error

	mu     sync.Mutex
	closed bool
	tasks  map[string]func()
}

// NewSwitch creates shutdown handler that triggers on any of provided OS signals
// and allows registered tasks to take up to provided timeout.
//
// When switch is triggered, tasks are invoked concurrently.
func NewSwitch(timeout time.Duration, signals ...os.Signal) *Switch {
	if signals == nil {
		signals = []os.Signal{syscall.SIGTERM, syscall.SIGINT}
	}

	done := make(chan error, 1)
	sh := &Switch{
		sig:   make(chan os.Signal, 1),
		tasks: make(map[string]func()),
		done:  done,
	}

	signal.Notify(sh.sig, signals...)

	go sh.waitForSignal(done, timeout)

	return sh
}

func (s *Switch) waitForSignal(done chan error, timeout time.Duration) {
	<-s.sig

	signal.Stop(s.sig)
	s.Shutdown()

	s.mu.Lock()

	sem := make(chan struct{}, len(s.tasks))
	active := make(map[string]struct{})

	for name, fn := range s.tasks {
		fn := fn
		name := name

		sem <- struct{}{}

		active[name] = struct{}{}

		go func() {
			defer func() {
				s.mu.Lock()
				delete(active, name)
				s.mu.Unlock()
				<-sem
			}()

			fn()
		}()
	}
	s.mu.Unlock()

	deadline := time.After(timeout)

	for i := 0; i < cap(sem); i++ {
		select {
		case sem <- struct{}{}:
		case <-deadline:
			var err ErrTimeout

			s.mu.Lock()

			for k := range active {
				err = append(err, k)
				sort.Strings(err)
			}

			s.mu.Unlock()

			done <- err
		}
	}

	close(done)
}

// Wait returns a channel that blocks until switch is triggered.
//
// Resulting channel may return a non-empty error if tasks fail to finish within a timeout.
func (s *Switch) Wait() <-chan error {
	return s.done
}

// OnShutdown adds a named task to run on shutdown.
func (s *Switch) OnShutdown(name string, fn func()) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tasks[name] = fn
}

// Shutdown triggers the switch and stops listening to OS signals.
func (s *Switch) Shutdown() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.closed {
		s.closed = true
		close(s.sig)
	}
}
