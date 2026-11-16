package example

import (
	"fmt"
	"log/slog"
)

type ctxerrPkg struct{}

func (ctxerrPkg) Wrap(err error, msg string) error { return err }

var ctxerr ctxerrPkg

func flagged(logger *slog.Logger, id uint) {
	logger.Info("moved host", "team_id", id)       // want `string uses deprecated "team"/"teams"`
	_ = fmt.Sprintf("Host belongs to team %d", id) // want `string uses deprecated "team"/"teams"`
	_ = "hostTeamID"                               // want `string uses deprecated "team"/"teams"`
}

func allowed(err error, id uint) error {
	_ = "fleet_id"
	_ = "TeamIdentifier"
	_ = "Send alerts to Microsoft Teams"
	_ = "SELECT name FROM teams WHERE id = ?"
	_ = ctxerr.Wrap(err, fmt.Sprintf("get team %d", id))
	return fmt.Errorf("load team: %w", err)
}
