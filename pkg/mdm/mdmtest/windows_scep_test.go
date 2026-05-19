package mdmtest

import (
	"context"
	"encoding/xml"
	"log/slog"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service/integrationtest/scep_server"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExtractSCEPCommandsAtomicLayout anchors the most-common parser shape — an Atomic-wrapped
// SCEP CSP with all required fields. Regressions surface here before they confuse the slow
// integration suite. The flat (non-Atomic) layout, /User locURI variant, and indented LocURI
// fixture are exercised by the integration suite directly via the existing profile fixtures.
func TestExtractSCEPCommandsAtomicLayout(t *testing.T) {
	atomic := buildAtomicSCEPCmd(t, "/Device", "uniq-1", "https://scep.example.com/scep", "challenge-abc",
		"CN=device-1,OU=fleet-1")
	cmds := map[string]fleet.ProtoCmdOperation{
		atomic.CmdID.Value: {Verb: fleet.CmdAtomic, Cmd: atomic},
	}

	c := &TestWindowsMDMClient{}
	got, incomplete := c.ExtractSCEPCommands(cmds)
	require.Empty(t, incomplete)
	require.Len(t, got, 1)
	assert.Equal(t, "uniq-1", got[0].UniqueID)
	assert.Equal(t, "https://scep.example.com/scep", got[0].ServerURL)
	assert.Equal(t, "challenge-abc", got[0].Challenge)
	assert.Equal(t, "CN=device-1,OU=fleet-1", got[0].SubjectName)
	assert.Equal(t, atomic.CmdID.Value, got[0].AtomicCmdID)
	assert.NotEmpty(t, got[0].EnrollCmdID)
}

// TestExtractSCEPCommandsSurfacesIncomplete guards the contract that AppendSCEPInstallResponses'
// hasWork return value depends on: malformed SCEP CSPs land in the incomplete slice instead of
// being silently dropped, so callers fail loudly. Without this, a future "drop incomplete" change
// would only surface as a confusing downstream failure.
func TestExtractSCEPCommandsSurfacesIncomplete(t *testing.T) {
	// Only Challenge is set: missing ServerURL, SubjectName, and the Enroll Exec.
	loc := "./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/incomplete/Install/Challenge"
	add := fleet.SyncMLCmd{
		XMLName: xml.Name{Local: fleet.CmdAdd},
		CmdID:   fleet.CmdID{Value: uuid.NewString()},
		Items:   []fleet.CmdItem{{Target: &loc, Data: &fleet.RawXmlData{Content: "ch"}}},
	}
	cmds := map[string]fleet.ProtoCmdOperation{
		add.CmdID.Value: {Verb: fleet.CmdAdd, Cmd: add},
	}
	complete, incomplete := (&TestWindowsMDMClient{}).ExtractSCEPCommands(cmds)
	assert.Empty(t, complete)
	require.Len(t, incomplete, 1)
	assert.Equal(t, "incomplete", incomplete[0].UniqueID)

	missing := missingSCEPFields(incomplete[0])
	assert.Contains(t, missing, "ServerURL")
	assert.Contains(t, missing, "SubjectName")
	assert.Contains(t, missing, "/Install/Enroll Exec")
}

// TestRunSCEPAgainstTestServer is the only fast end-to-end SCEP exchange test in the codebase.
// If RunSCEP, performSCEPExchange, or the test SCEP server's in-memory signer breaks, this
// fires in ~50ms instead of waiting for the MYSQL_TEST integration suite.
func TestRunSCEPAgainstTestServer(t *testing.T) {
	srv := scep_server.StartTestSCEPServer(t)

	c := &TestWindowsMDMClient{}
	cert, err := c.RunSCEP(context.Background(), SCEPCommand{
		UniqueID:    "test",
		ServerURL:   srv.URL + "/scep",
		Challenge:   "any-challenge",
		SubjectName: "CN=integration-test,OU=fleet",
		KeyLength:   2048,
	}, slog.New(slog.DiscardHandler))
	require.NoError(t, err)
	require.NotNil(t, cert)
	assert.Equal(t, "integration-test", cert.Subject.CommonName)
	assert.True(t, strings.HasPrefix(cert.Subject.OrganizationalUnit[0], "fleet"))
}

// buildAtomicSCEPCmd constructs an Atomic command containing the typical SCEP CSP nodes Fleet
// sends, plus an Exec on /Install/Enroll. locPrefix should be either "/Device" or "/User".
func buildAtomicSCEPCmd(t *testing.T, locPrefix, uniqueID, serverURL, challenge, subject string) fleet.SyncMLCmd {
	t.Helper()
	mkAdd := func(field, data string) fleet.SyncMLCmd {
		loc := "." + locPrefix + "/Vendor/MSFT/ClientCertificateInstall/SCEP/" + uniqueID + "/Install/" + field
		return fleet.SyncMLCmd{
			XMLName: xml.Name{Local: fleet.CmdAdd},
			CmdID:   fleet.CmdID{Value: uuid.NewString()},
			Items:   []fleet.CmdItem{{Target: &loc, Data: &fleet.RawXmlData{Content: data}}},
		}
	}
	enrollLoc := "." + locPrefix + "/Vendor/MSFT/ClientCertificateInstall/SCEP/" + uniqueID + "/Install/Enroll"
	return fleet.SyncMLCmd{
		XMLName: xml.Name{Local: fleet.CmdAtomic},
		CmdID:   fleet.CmdID{Value: uuid.NewString()},
		AddCommands: []fleet.SyncMLCmd{
			mkAdd("ServerURL", serverURL),
			mkAdd("Challenge", challenge),
			mkAdd("SubjectName", subject),
			mkAdd("KeyLength", "2048"),
			mkAdd("HashAlgorithm", "SHA-256"),
		},
		ExecCommands: []fleet.SyncMLCmd{{
			XMLName: xml.Name{Local: fleet.CmdExec},
			CmdID:   fleet.CmdID{Value: uuid.NewString()},
			Items:   []fleet.CmdItem{{Target: &enrollLoc}},
		}},
	}
}
