package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

const maxRetries = 5

// Job defines an interface for jobs that can be run by the Worker
type Job interface {
	// Name is the unique name of the job.
	Name() string

	// Run performs the actual work.
	Run(ctx context.Context, argsJSON json.RawMessage) error
}

// Worker runs jobs. NOT SAFE FOR CONCURRENT USE.
type Worker struct {
	ds  fleet.Datastore
	log kitlog.Logger

	registry map[string]Job
}

func NewWorker(ds fleet.Datastore, log kitlog.Logger) *Worker {
	return &Worker{
		ds:       ds,
		log:      log,
		registry: make(map[string]Job),
	}
}

func (w *Worker) Register(jobs ...Job) {
	for _, j := range jobs {
		name := j.Name()
		if _, ok := w.registry[name]; ok {
			panic(fmt.Sprintf("job %s already registered", name))
		}
		w.registry[name] = j
	}
}

func QueueJob(ctx context.Context, ds fleet.Datastore, name string, args interface{}) (*fleet.Job, error) {
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("marshal args: %w", err)
	}
	job := &fleet.Job{
		Name:  name,
		Args:  (*json.RawMessage)(&argsJSON),
		State: fleet.JobStateQueued,
	}

	return ds.NewJob(ctx, job)
}

// ProcessJobs processes all queued jobs.
func (w *Worker) ProcessJobs(ctx context.Context) error {

	// process jobs until there are none left
	maxNumJobs := 100
	for {
		jobs, err := w.ds.GetQueuedJobs(ctx, maxNumJobs)
		if err != nil {
			return fmt.Errorf("get jobs: %w", err)
		}

		if len(jobs) == 0 {
			break
		}

		for _, job := range jobs {
			log := kitlog.With(w.log, "job_id", job.ID)

			level.Debug(log).Log("msg", "processing job")

			err := w.processJob(ctx, job)
			if err != nil {
				level.Error(log).Log("msg", "job failed", "err", err)
				job.Error = err.Error()
				if job.Retries < maxRetries {
					level.Debug(log).Log("msg", "retrying job")
					job.Retries += 1
				} else {
					job.State = fleet.JobStateFailure
				}
			} else {
				job.State = fleet.JobStateSuccess
				job.Error = ""
			}

			_, err = w.ds.UpdateJob(ctx, job.ID, job)
			if err != nil {
				level.Error(log).Log("update job", "err", err)
			}
		}
	}

	return nil
}

func (w *Worker) processJob(ctx context.Context, job *fleet.Job) error {
	j, ok := w.registry[job.Name]
	if !ok {
		return fmt.Errorf("unknown job: %s", job.Name)
	}

	var args json.RawMessage
	if job.Args != nil {
		args = *job.Args
	}

	return j.Run(ctx, args)
}
