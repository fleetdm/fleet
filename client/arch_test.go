package client

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/archtest"
)

const m = archtest.ModuleName

func TestClientPackageDoesNotImportServerService(t *testing.T) {
	t.Parallel()
	archtest.NewPackageTest(t, m+"/client...").
		ShouldNotDependOn(
			m+"/server/service...",
			m+"/ee/server/service...",
		).
		IgnoreDeps(
			m + "/server/service/externalsvc", // server/fleet has a dependency on Jira and Zendesk.
		).
		Check()
}
