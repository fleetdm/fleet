package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

type ctxKey int

const (
	maxRetries = 5
	// nvdCVEURL is the base link to a CVE on the NVD website, only the CVE code
	// needs to be appended to make it a valid link.
	nvdCVEURL = "https://nvd.nist.gov/vuln/detail/"

	// context key for the retry number of a job, made available via the context
	// to the job processor.
	retryNumberCtxKey = ctxKey(0)
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
	PolicyID       uint                  `json:"policy_id"`
	PolicyName     string                `json:"policy_name"`
	PolicyCritical bool                  `json:"policy_critical"`
	Hosts          []fleet.PolicySetHost `json:"hosts"`
	TeamID         *uint                 `json:"team_id,omitempty"`
}

// vulnArgs are the args common to all integrations that can process
// vulnerabilities.
type vulnArgs struct {
	CVE                 string     `json:"cve,omitempty"`
	AffectedSoftwareIDs []uint     `json:"affected_software,omitempty"`
	EPSSProbability     *float64   `json:"epss_probability,omitempty"`   // Premium feature only
	CVSSScore           *float64   `json:"cvss_score,omitempty"`         // Premium feature only
	CISAKnownExploit    *bool      `json:"cisa_known_exploit,omitempty"` // Premium feature only
	CVEPublished        *time.Time `json:"cve_published,omitempty"`      // Premium feature only
}

// Worker runs jobs. NOT SAFE FOR CONCURRENT USE.
type Worker struct {
	ds  fleet.Datastore
	log kitlog.Logger

	// For tests only, allows ignoring unknown jobs instead of failing them.
	TestIgnoreUnknownJobs bool

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
	return QueueJobWithDelay(ctx, ds, name, args, 0)
}

// QueueJobWithDelay is like QueueJob but does not make the job available
// before a specified delay (or no delay if delay is <= 0).
func QueueJobWithDelay(ctx context.Context, ds fleet.Datastore, name string, args interface{}, delay time.Duration) (*fleet.Job, error) {
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "marshal args")
	}

	var notBefore time.Time
	if delay > 0 {
		notBefore = time.Now().UTC().Add(delay)
	}
	job := &fleet.Job{
		Name:      name,
		Args:      (*json.RawMessage)(&argsJSON),
		State:     fleet.JobStateQueued,
		NotBefore: notBefore,
	}

	return ds.NewJob(ctx, job)
}

// this defines the delays to add between retries (i.e. how the "not_before"
// timestamp of a job will be set for the next run). Keep in mind that at a
// minimum, the job will not be retried before the next cron run of the worker,
// but we want to ensure a minimum delay before retries to give a chance to
// e.g. transient network issues to resolve themselves.
var delayPerRetry = []time.Duration{
	1: 0, // i.e. for the first retry, do it ASAP (on the next worker run)
	2: 5 * time.Minute,
	3: 10 * time.Minute,
	4: 1 * time.Hour,
	5: 2 * time.Hour,
}

// ProcessJobs processes all queued jobs.
func (w *Worker) ProcessJobs(ctx context.Context) error {
	const maxNumJobs = 100

	// process jobs until there are none left or the context is cancelled
	seen := make(map[uint]struct{})
	for {
		jobs, err := w.ds.GetQueuedJobs(ctx, maxNumJobs, time.Time{})
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
					if job.Retries < len(delayPerRetry) {
						job.NotBefore = time.Now().Add(delayPerRetry[job.Retries])
					}
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
		if w.TestIgnoreUnknownJobs {
			return nil
		}
		return ctxerr.Errorf(ctx, "unknown job: %s", job.Name)
	}

	var args json.RawMessage
	if job.Args != nil {
		args = *job.Args
	}

	ctx = context.WithValue(ctx, retryNumberCtxKey, job.Retries)
	return j.Run(ctx, args)
}

type failingPoliciesTplArgs struct {
	FleetURL       string
	PolicyID       uint
	PolicyName     string
	PolicyCritical bool
	TeamID         *uint
	Hosts          []fleet.PolicySetHost
}

func newFailingPoliciesTplArgs(fleetURL string, args *failingPolicyArgs) *failingPoliciesTplArgs {
	return &failingPoliciesTplArgs{
		FleetURL:       fleetURL,
		PolicyName:     args.PolicyName,
		PolicyID:       args.PolicyID,
		PolicyCritical: args.PolicyCritical,
		TeamID:         args.TeamID,
		Hosts:          args.Hosts,
	}
}
