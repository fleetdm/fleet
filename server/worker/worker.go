package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

const (
	maxRetries = 5
	// nvdCVEURL is the base link to a CVE on the NVD website, only the CVE code
	// needs to be appended to make it a valid link.
	nvdCVEURL = "https://nvd.nist.gov/vuln/detail/"
)

const (
	// types of integrations - jobs like Jira and Zendesk support different
	// integrations, this identifies the integration type of a message.
	intgTypeVuln          = "vuln"
	intgTypeFailingPolicy = "failingPolicy"
)

// Job defines an interface for jobs that can be run by the Worker
type Job interface {
	// Name is the unique name of the job.
	Name() string

	// Run performs the actual work.
	Run(ctx context.Context, argsJSON json.RawMessage) error
}

// failingPolicyArgs are the args common to all integrations that can process
// failing policies.
type failingPolicyArgs struct {
	PolicyID   uint                  `json:"policy_id"`
	PolicyName string                `json:"policy_name"`
	Hosts      []fleet.PolicySetHost `json:"hosts"`
	TeamID     *uint                 `json:"team_id,omitempty"`
}

// vulnArgs are the args common to all integrations that can process
// vulnerabilities.
type vulnArgs struct {
	CVE              string   `json:"cve,omitempty"`
	EPSSProbability  *float64 `json:"epss_probability,omitempty"`   // Premium feature only
	CVSSScore        *float64 `json:"cvss_score,omitempty"`         // Premium feature only
	CISAKnownExploit *bool    `json:"cisa_known_exploit,omitempty"` // Premium feature only
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

// QueueJob inserts a job to be processed by the worker for the job processor
// identified by the name (e.g. "jira"). The args value is marshaled as JSON
// and provided to the job processor when the job is executed.
func QueueJob(ctx context.Context, ds fleet.Datastore, name string, args interface{}) (*fleet.Job, error) {
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "marshal args")
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
	const maxNumJobs = 100

	// process jobs until there are none left or the context is cancelled
	seen := make(map[uint]struct{})
	for {
		jobs, err := w.ds.GetQueuedJobs(ctx, maxNumJobs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "get queued jobs")
		}

		if len(jobs) == 0 {
			break
		}

		for _, job := range jobs {
			select {
			case <-ctx.Done():
				return ctxerr.Wrap(ctx, ctx.Err(), "context done")
			default:
			}

			log := kitlog.With(w.log, "job_id", job.ID)

			if _, ok := seen[job.ID]; ok {
				level.Debug(log).Log("msg", "some jobs failed, retrying on next cron execution")
				return nil
			}
			seen[job.ID] = struct{}{}

			level.Debug(log).Log("msg", "processing job")

			if err := w.processJob(ctx, job); err != nil {
				level.Error(log).Log("msg", "process job", "err", err)
				job.Error = err.Error()
				if job.Retries < maxRetries {
					level.Debug(log).Log("msg", "will retry job")
					job.Retries += 1
				} else {
					job.State = fleet.JobStateFailure
				}
			} else {
				job.State = fleet.JobStateSuccess
				job.Error = ""
			}

			// When we update the job, the updated_at timestamp gets updated and the job gets "pushed" to the back
			// of queue. GetQueuedJobs fetches jobs by updated_at, so it will not return the same job until the queue
			// has been processed once.
			if _, err := w.ds.UpdateJob(ctx, job.ID, job); err != nil {
				level.Error(log).Log("update job", "err", err)
			}
		}
	}

	return nil
}

func (w *Worker) processJob(ctx context.Context, job *fleet.Job) error {
	j, ok := w.registry[job.Name]
	if !ok {
		return ctxerr.Errorf(ctx, "unknown job: %s", job.Name)
	}

	var args json.RawMessage
	if job.Args != nil {
		args = *job.Args
	}

	return j.Run(ctx, args)
}
