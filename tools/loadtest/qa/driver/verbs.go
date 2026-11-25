package main

import (
	"context"
	"log/slog"
	"sync"
)

// result is what one run of an action returns. status is the HTTP status of the action's main request (0 if it
// never got one); skip says why the action had nothing to do (no online host, an empty pool), which isn't a failure;
// slow marks an outcome that came late even though the request itself worked.
type result struct {
	status int
	err    error
	skip   string
	slow   bool
	detail map[string]any
}

// verb is one kind of admin operation. A spec builds its own verb from its `with` block; setup runs once before
// the first run, and run must honor ctx.
type verb interface {
	setup(ctx context.Context, e *env) error
	run(ctx context.Context, e *env) result
}

type verbFactory func(p *params) (verb, error)

var verbs = map[string]verbFactory{
	"live_query":           newLiveQuery,
	"users.login":          newUsersLogin,
	"hosts.idp":            newHostsIDP,
	"labels.membership":    newLabelsMembership,
	"gitops.apply":         newGitopsApply,
	"fleets.agent_options": newFleetsAgentOptions,
	"software.uninstall":   newSoftwareUninstall,
	"profiles.resend":      newProfilesResend,
	"hosts.transfer":       newHostsTransfer,
}

// env is what verbs share: the instance, its host list and the driver's own objects.
type env struct {
	api      *api
	url      string
	insecure bool
	stateDir string
	hosts    *hostCache
	logger   *slog.Logger

	userPrefix string
	userCount  int
	usersOnce  sync.Once
	users      *userPool
	usersErr   error
}
