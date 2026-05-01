package mdmtest

import (
	"context"
	"encoding/xml"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service/integrationtest/scep_server"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractSCEPCommandsAtomicLayout(t *testing.T) {
	atomic := buildAtomicSCEPCmd(t, "/Device", "uniq-1", "https://scep.example.com/scep", "challenge-abc",
		"CN=device-1,OU=fleet-1")
	cmds := map[string]fleet.ProtoCmdOperation{
		atomic.CmdID.Value: {Verb: fleet.CmdAtomic, Cmd: atomic},
	}

	c := &TestWindowsMDMClient{}
	got := c.ExtractSCEPCommands(cmds)
	require.Len(t, got, 1)
	assert.Equal(t, "uniq-1", got[0].UniqueID)
	assert.Equal(t, "https://scep.example.com/scep", got[0].ServerURL)
	assert.Equal(t, "challenge-abc", got[0].Challenge)
	assert.Equal(t, "CN=device-1,OU=fleet-1", got[0].SubjectName)
	assert.Equal(t, atomic.CmdID.Value, got[0].AtomicCmdID)
	assert.NotEmpty(t, got[0].EnrollCmdID)
}

func TestExtractSCEPCommandsUserLocURI(t *testing.T) {
	atomic := buildAtomicSCEPCmd(t, "/User", "user-uniq", "https://scep.example.com/scep", "challenge",
		"CN=u")
	cmds := map[string]fleet.ProtoCmdOperation{
		atomic.CmdID.Value: {Verb: fleet.CmdAtomic, Cmd: atomic},
	}
	got := (&TestWindowsMDMClient{}).ExtractSCEPCommands(cmds)
	require.Len(t, got, 1)
	assert.Equal(t, "user-uniq", got[0].UniqueID)
}

func TestExtractSCEPCommandsFlatLayout(t *testing.T) {
	add := func(field, data string) fleet.SyncMLCmd {
		loc := "./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/uniq-2/Install/" + field
		return fleet.SyncMLCmd{
			XMLName: xml.Name{Local: fleet.CmdAdd},
			CmdID:   fleet.CmdID{Value: uuid.NewString()},
			Items:   []fleet.CmdItem{{Target: &loc, Data: &fleet.RawXmlData{Content: data}}},
		}
	}
	exec := func() fleet.SyncMLCmd {
		loc := "./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/uniq-2/Install/Enroll"
		return fleet.SyncMLCmd{
			XMLName: xml.Name{Local: fleet.CmdExec},
			CmdID:   fleet.CmdID{Value: uuid.NewString()},
			Items:   []fleet.CmdItem{{Target: &loc}},
		}
	}

	addServerURL := add("ServerURL", "https://scep.example.com/scep")
	addChallenge := add("Challenge", "ch")
	addSubject := add("SubjectName", "CN=foo")
	execEnroll := exec()

	cmds := map[string]fleet.ProtoCmdOperation{
		addServerURL.CmdID.Value: {Verb: fleet.CmdAdd, Cmd: addServerURL},
		addChallenge.CmdID.Value: {Verb: fleet.CmdAdd, Cmd: addChallenge},
		addSubject.CmdID.Value:   {Verb: fleet.CmdAdd, Cmd: addSubject},
		execEnroll.CmdID.Value:   {Verb: fleet.CmdExec, Cmd: execEnroll},
	}
	got := (&TestWindowsMDMClient{}).ExtractSCEPCommands(cmds)
	require.Len(t, got, 1)
	assert.Empty(t, got[0].AtomicCmdID, "flat layout should not stamp an Atomic CmdID")
	assert.Equal(t, execEnroll.CmdID.Value, got[0].EnrollCmdID)
	assert.Len(t, got[0].AddCmdIDs, 3)
}

func TestExtractSCEPCommandsIgnoresIncomplete(t *testing.T) {
	// Missing ServerURL: should be dropped.
	loc := "./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/incomplete/Install/Challenge"
	add := fleet.SyncMLCmd{
		XMLName: xml.Name{Local: fleet.CmdAdd},
		CmdID:   fleet.CmdID{Value: uuid.NewString()},
		Items:   []fleet.CmdItem{{Target: &loc, Data: &fleet.RawXmlData{Content: "ch"}}},
	}
	cmds := map[string]fleet.ProtoCmdOperation{
		add.CmdID.Value: {Verb: fleet.CmdAdd, Cmd: add},
	}
	got := (&TestWindowsMDMClient{}).ExtractSCEPCommands(cmds)
	assert.Empty(t, got)
}

func TestSCEPInstallPathHandlesIndentedLocURI(t *testing.T) {
	// Fleet test fixtures sometimes have leading whitespace inside the LocURI text node.
	indented := "\n                ./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/abc/Install/ServerURL"
	p, ok := scepInstallPath(indented)
	require.True(t, ok)
	assert.Equal(t, "abc", p.UniqueID)
	assert.Equal(t, "ServerURL", p.Field)
}

func TestSendResponseFlushesPendingAlertWhenQueueEmpty(t *testing.T) {
	c := &TestWindowsMDMClient{}
	c.queueSCEPAlert(SCEPCommand{UniqueID: "abc"}, nil)

	// Mimic what AppendSCEPInstallResponses needs: pendingAlerts present, queuedCommandResponses empty.
	// SendResponse usually requires lastManagementResp to be populated; we only call takePendingAlerts
	// here to confirm the queue contents survive until drain time.
	got := c.takePendingAlerts()
	require.Len(t, got, 1)
	assert.Equal(t, fleet.CmdAlert, got[0].XMLName.Local)
	require.Len(t, got[0].Items, 1)
	require.NotNil(t, got[0].Items[0].Data)
	assert.Contains(t, got[0].Items[0].Data.Content, `"unique_id":"abc"`)
	assert.Contains(t, got[0].Items[0].Data.Content, `"status":"success"`)

	// Subsequent calls return empty.
	assert.Empty(t, c.takePendingAlerts())
}

func TestQueueSCEPAlertReportsFailureStatus(t *testing.T) {
	c := &TestWindowsMDMClient{}
	c.queueSCEPAlert(SCEPCommand{UniqueID: "xyz"}, errors.New("boom"))
	alerts := c.takePendingAlerts()
	require.Len(t, alerts, 1)
	assert.Contains(t, alerts[0].Items[0].Data.Content, `"status":"failure"`)
}

func TestParseSCEPSubject(t *testing.T) {
	cases := []struct {
		in     string
		cn     string
		ouLen  int
		errMsg string
	}{
		{in: "CN=foo,OU=bar", cn: "foo", ouLen: 1},
		{in: " CN = a , OU = b , OU = c ", cn: "a", ouLen: 2},
		{in: "OU=only", cn: "", ouLen: 1},
		{in: "", errMsg: "empty subject"},
		{in: "garbage", errMsg: "no recognized RDN"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got, err := parseSCEPSubject(tc.in)
			if tc.errMsg != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errMsg)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.cn, got.CommonName)
			assert.Len(t, got.OrganizationalUnit, tc.ouLen)
		})
	}
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

func TestRunSCEPAgainstTestServer(t *testing.T) {
	srv := scep_server.StartTestSCEPServer(t)

	c := &TestWindowsMDMClient{}
	cert, err := c.RunSCEP(context.Background(), SCEPCommand{
		UniqueID:    "test",
		ServerURL:   srv.URL + "/scep",
		Challenge:   "any-challenge",
		SubjectName: "CN=integration-test,OU=fleet",
		KeyLength:   2048,
		HashAlg:     "SHA-256",
	}, slog.New(slog.DiscardHandler))
	require.NoError(t, err)
	require.NotNil(t, cert)
	assert.Equal(t, "integration-test", cert.Subject.CommonName)
	assert.True(t, strings.HasPrefix(cert.Subject.OrganizationalUnit[0], "fleet"))
}
