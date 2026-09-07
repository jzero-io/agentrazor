package agent

import (
	"errors"
	"sync/atomic"
	"testing"
)

type closeTrackingRuntime struct {
	ThreadRuntime
	closed atomic.Bool
}

func (r *closeTrackingRuntime) Close() error {
	r.closed.Store(true)
	return nil
}

func TestRestartRuntimeClosesOldRuntimeBeforeStartingNewOne(t *testing.T) {
	old := &closeTrackingRuntime{}
	next := &closeTrackingRuntime{}
	service := NewThreadService(old, func() (ThreadRuntime, error) {
		if !old.closed.Load() {
			return nil, errors.New("old runtime is still running")
		}
		return next, nil
	})
	defer func() { _ = service.Close() }()

	if err := service.RestartRuntime(); err != nil {
		t.Fatal(err)
	}
	if !old.closed.Load() {
		t.Fatal("old runtime was not closed")
	}
	if service.runtime != next {
		t.Fatal("new runtime was not installed")
	}
}
