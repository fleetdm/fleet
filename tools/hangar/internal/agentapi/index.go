package agentapi

import (
	"net/http"

	"github.com/fleetdm/fleet/tools/hangar/internal/slots"
)

type indexRoute struct {
	Method string `json:"method"`
	Path   string `json:"path"`
	Does   string `json:"does"`
}

type indexAction struct {
	Name string   `json:"name"`
	Body []string `json:"body"`
	Does string   `json:"does"`
}

// apiIndex lets an agent discover the API from the socket alone.
var apiIndex = struct {
	Routes  []indexRoute      `json:"routes"`
	Actions []indexAction     `json:"actions"`
	Flow    []string          `json:"flow"`
	Cleanup []string          `json:"cleanup"`
	Status  map[string]string `json:"status"`
}{
	Routes: []indexRoute{
		{"GET", "/v1", "This index."},
		{"GET", "/v1/slots", "Slots with worktree, ports, compose project, reservation owner, holder, and process states."},
		{"POST", "/v1/slots/{id}/actions", "Runs an action. reserve, assign, and release return 200; the rest return 202 with a job."},
		{"GET", "/v1/jobs/{id}", "Job state: running, succeeded, failed, or interrupted."},
		{"GET", "/v1/jobs/{id}/log?since_ms=", "The job's log lines, with tokens and passwords scrubbed."},
	},
	Actions: []indexAction{
		{"reserve", []string{"worktree"}, "Reserves the slot for worktree, recording the slot's current worktree as previous_worktree. Repeating it for the same worktree is a no-op; another worktree's reservation returns 409."},
		{"assign", []string{"worktree", "target"}, "Sets the slot's worktree to target: the reservation's worktree or previous_worktree, or null when previous_worktree is null. The slot must be idle."},
		{slots.ActionBuild, []string{"worktree"}, "make deps, make generate, make build."},
		{slots.ActionDockerUp, []string{"worktree"}, "compose down, then up -d mysql redis s3."},
		{slots.ActionPrepareDB, []string{"worktree"}, "Backs up a non-empty database, then runs fleet prepare db. With a docker-compose.override.yml in the worktree, MySQL must use a volume it names."},
		{slots.ActionServe, []string{"worktree"}, "Replaces a running fleet serve and succeeds once it listens."},
		{slots.ActionStop, []string{"worktree"}, "Stops every slot process, then compose down without removing volumes."},
		{slots.ActionBackup, []string{"worktree", "name"}, "Dumps the database to db-backups/<slot>/<name>.sql.gz. Runs beside serve; never overwrites."},
		{"release", []string{"worktree"}, "Drops your reservation. The slot must be idle and assigned back to previous_worktree."},
	},
	Flow:    []string{"reserve", "assign (target = your worktree)", "build", "docker-up", "prepare-db", "serve", "GET /healthz on the slot's ports.server"},
	Cleanup: []string{"stop", "assign (target = previous_worktree)", "release"},
	Status: map[string]string{
		"403": "A guard refused: s1, a port shared with s1, no reservation, or a worktree mismatch.",
		"409": "The slot is busy or taken: a job, a GUI process, a Docker operation, another worktree's reservation, a release before assigning back, or an existing backup name.",
		"503": "Hangar is shutting down.",
	},
}

func (a *API) index(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, apiIndex)
}
