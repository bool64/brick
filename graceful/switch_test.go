package graceful_test

import (
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/acme-corp-tech/brick/graceful"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewShutdown_default_SIGINT(t *testing.T) {
	var ok bool

	done := graceful.NewSwitch(time.Minute)
	done.OnShutdown("test", func() { ok = true })

	require.NoError(t, syscall.Kill(os.Getpid(), syscall.SIGINT))

	select {
	case <-done.Wait():
	case <-time.After(time.Second):
		assert.Fail(t, "failed to shutdown in reasonable time")
	}

	assert.True(t, ok)
}

func TestNewShutdown_default_SIGTERM(t *testing.T) {
	var ok bool

	done := graceful.NewSwitch(time.Minute)
	done.OnShutdown("test", func() { ok = true })

	require.NoError(t, syscall.Kill(os.Getpid(), syscall.SIGTERM))

	select {
	case <-done.Wait():
	case <-time.After(time.Second):
		assert.Fail(t, "failed to shutdown in reasonable time")
	}

	assert.True(t, ok)
}

func TestNewShutdown_custom(t *testing.T) {
	var ok bool

	done := graceful.NewSwitch(time.Minute, syscall.SIGHUP)
	done.OnShutdown("test", func() { ok = true })
	require.NoError(t, syscall.Kill(os.Getpid(), syscall.SIGHUP))

	select {
	case <-done.Wait():
	case <-time.After(time.Second):
		assert.Fail(t, "failed to shutdown in reasonable time")
	}

	assert.True(t, ok)
}

func TestNewShutdown_manual(t *testing.T) {
	var ok bool

	done := graceful.NewSwitch(time.Minute, syscall.SIGTERM)
	done.OnShutdown("test", func() { ok = true })
	done.Shutdown()

	select {
	case <-done.Wait():
	case <-time.After(time.Second):
		assert.Fail(t, "failed to shutdown in reasonable time")
	}

	assert.True(t, ok)
	assert.NotPanics(t, done.Shutdown)
}

func TestNewShutdown_timeout(t *testing.T) {
	done := graceful.NewSwitch(time.Millisecond)
	done.OnShutdown("test1", func() { time.Sleep(time.Minute) })
	done.OnShutdown("test2", func() { time.Sleep(time.Minute) })
	done.OnShutdown("test3", func() {})
	done.Shutdown()

	select {
	case err := <-done.Wait():
		assert.EqualError(t, err, "shutdown timeout, tasks left: test1, test2")
	case <-time.After(time.Second):
		assert.Fail(t, "failed to shutdown in reasonable time")
	}
}
