package controller

import (
	"sync"
	"testing"
	"time"
)

func TestNormalizeModelTestConcurrency(t *testing.T) {
	if got := normalizeModelTestConcurrency(0); got != 3 {
		t.Fatalf("default=%d", got)
	}
	if got := normalizeModelTestConcurrency(5); got != 5 {
		t.Fatalf("five=%d", got)
	}
	if got := normalizeModelTestConcurrency(99); got != 10 {
		t.Fatalf("cap=%d", got)
	}
}

func TestChannelModelTestJobPauseResumeCancel(t *testing.T) {
	job := &channelModelTestJob{
		Running: true,
	}
	job.cond = sync.NewCond(&job.mu)

	if !job.pause() {
		t.Fatal("pause should change state")
	}
	if !job.Paused {
		t.Fatal("job should be paused")
	}
	if job.pause() {
		t.Fatal("pause twice should be ignored")
	}

	waitDone := make(chan bool, 1)
	go func() {
		waitDone <- job.waitForDispatch()
	}()
	select {
	case <-waitDone:
		t.Fatal("waitForDispatch should block while paused")
	case <-time.After(80 * time.Millisecond):
	}

	if !job.resume() {
		t.Fatal("resume should change state")
	}
	select {
	case ok := <-waitDone:
		if !ok {
			t.Fatal("resumed dispatch should be allowed")
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("waitForDispatch did not unblock after resume")
	}

	if !job.cancel() {
		t.Fatal("cancel should change state")
	}
	if job.waitForDispatch() {
		t.Fatal("dispatch should stop after cancel")
	}
}
