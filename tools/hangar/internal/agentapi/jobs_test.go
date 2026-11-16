package agentapi

import (
	"context"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// blockingDown blocks compose down until release closes.
type blockingDown struct {
	*fakeManager
	once    sync.Once
	entered chan struct{}
	release chan struct{}
}

func (d *blockingDown) ComposeDown(id, cwd, project, owner string) error {
	d.once.Do(func() { close(d.entered) })
	<-d.release
	return nil
}

// ctxDown is blockingDown that also ends when the job's ctx does.
type ctxDown struct{ *blockingDown }

func (d ctxDown) ComposeDownContext(ctx context.Context, id, cwd, project, owner string) error {
	d.once.Do(func() { close(d.entered) })
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-d.release:
		return nil
	}
}

// stopJobInComposeDown runs a stop job on an env whose manager wrap builds and
// returns once the job is inside compose down.
func stopJobInComposeDown(t *testing.T, wrap func(*blockingDown) Manager) (*env, *blockingDown, string) {
	t.Helper()
	e := newEnv(t)
	d := &blockingDown{fakeManager: e.mgr, entered: make(chan struct{}), release: make(chan struct{})}
	t.Cleanup(func() { close(d.release) }) // before newEnv's Close
	opts := e.api.opts
	opts.Manager = wrap(d)
	e.api = New(opts)
	e.srv.Close()
	e.srv = httptest.NewServer(e.api.Handler())
	e.assignTask()

	code, j := e.act("s2", map[string]any{"action": "stop"})
	if code != 202 {
		t.Fatalf("stop: %d %v", code, j)
	}
	select {
	case <-d.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("stop job never reached compose down")
	}
	return e, d, j["id"].(string)
}

func closeWithin(t *testing.T, api *API, limit time.Duration) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		api.Close()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(limit):
		t.Fatalf("Close still blocked after %v", limit)
	}
}

func TestCloseCancelsComposeDown(t *testing.T) {
	e, _, id := stopJobInComposeDown(t, func(d *blockingDown) Manager { return ctxDown{d} })
	closeWithin(t, e.api, 3*time.Second)
	if got := e.waitJob(id); got["state"] != stateInterrupted {
		t.Fatalf("job = %v", got)
	}
}

func TestCloseIsBoundedWhenAStepIgnoresCancel(t *testing.T) {
	prev := closeWait
	closeWait = 200 * time.Millisecond
	t.Cleanup(func() { closeWait = prev })
	e, _, _ := stopJobInComposeDown(t, func(d *blockingDown) Manager { return d })
	closeWithin(t, e.api, 3*time.Second)
}
