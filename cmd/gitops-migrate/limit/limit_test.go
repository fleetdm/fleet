package limit

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLimit(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		t.Parallel()

		// Init a limiter.
		l := New(2)

		// Wrap the testing context with a cancellation.
		ctx, cancel := context.WithCancel(t.Context())
		// Spawn two concurrent jobs (these will block until context cancel).
		l.Go(testBlocker(ctx))
		l.Go(testBlocker(ctx))

		// Wait brieeeeefly to avoid a race in G spawn.
		<-time.After(250 * time.Millisecond)

		// Ensure we see the expected number of concurrent jobs running.
		require.Equal(t, int32(2), l.jobsConcurrent.Load())

		// Cancel the context, ensure we see the job count head back to 0.
		cancel()

		// Wait brieeeeefly to avoid a race in G exit.
		<-time.After(250 * time.Millisecond)

		// Expect 0 concurrent jobs, 0 queued jobs.
		require.Equal(t, int32(0), l.jobsConcurrent.Load())
		require.Equal(t, int32(0), l.jobsQueued.Load())

		require.NoError(t, l.WaitContext(t.Context()))
	})
	t.Run("wait-context-deadlines", func(t *testing.T) {
		t.Parallel()

		// Init a limiter.
		l := New(2)

		// Wrap the testing context with a cancellation and a timeout.
		ctx, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
		l.Go(testBlocker(ctx))
		l.Go(testBlocker(ctx))

		// Expect a deadline exceeded error.
		require.Error(t, l.WaitContext(ctx))
		cancel()

		require.NoError(t, l.WaitContext(t.Context()))
	})
	t.Run("queued-jobs-stall-until-queue-frees-up", func(t *testing.T) {
		t.Parallel()

		// Here we test
		l := New(2)

		// Wrap the testing context with a cancellation. We use two contexts to
		// simulate "waves" of jobs.
		ctx1, cancel1 := context.WithCancel(t.Context())
		ctx2, cancel2 := context.WithCancel(t.Context())

		// Spawn two jobs.
		l.Go(testBlocker(ctx1))
		l.Go(testBlocker(ctx1))

		// Wait brieeeeefly to avoid a race in setup.
		<-time.After(250 * time.Millisecond)

		// Spawn two more jobs.
		l.Go(testBlocker(ctx2))
		l.Go(testBlocker(ctx2))

		// Wait brieeeeefly to avoid a race in setup.
		<-time.After(250 * time.Millisecond)

		// Expect two 2 jobs queued and 2 jobs active.
		require.Equal(t, int32(2), l.jobsQueued.Load())
		require.Equal(t, int32(2), l.jobsConcurrent.Load())

		// Cancel the first wave of jobs.
		cancel1()

		// Wait brieeeeefly to avoid a race in G exit.
		<-time.After(250 * time.Millisecond)

		// Expect two 0 jobs queued and 2 jobs active.
		require.Equal(t, int32(0), l.jobsQueued.Load())
		require.Equal(t, int32(2), l.jobsConcurrent.Load())

		// Cancel the second wave of jobs.
		cancel2()

		// Wait brieeeeefly to avoid a race in G exit.
		<-time.After(250 * time.Millisecond)

		// Expect two 0 jobs queued and 0 jobs active.
		require.Equal(t, int32(0), l.jobsQueued.Load())
		require.Equal(t, int32(0), l.jobsConcurrent.Load())

		// Wait for all jobs to complete
		l.Wait()
	})
}

func testBlocker(ctx context.Context) func() {
	return func() {
		<-ctx.Done()
	}
}
