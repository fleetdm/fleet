package notarize

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/require"
)

func init() {
	childCommands["log-accepted"] = testCmdLogValidSubmission
	childCommands["log-invalid"] = testCmdLogInvalidSubmission
}

func TestLog_accepted(t *testing.T) {
	log, err := log(context.Background(), "foo", &Options{
		Logger:  hclog.L(),
		BaseCmd: childCmd(t, "log-accepted"),
	})

	require := require.New(t)
	require.NoError(err)
	require.Equal(log.JobId, "3382aa04-e417-46a0-b1b4-42eebf85906c")
	require.Equal(log.Status, "Accepted")
	require.Equal(log.StatusSummary, "Ready for distribution")
	require.Equal(len(log.Issues), 0)
	require.Equal(len(log.TicketContents), 1)
}

func TestLog_invalid(t *testing.T) {
	log, err := log(context.Background(), "foo", &Options{
		Logger:  hclog.L(),
		BaseCmd: childCmd(t, "log-invalid"),
	})

	require := require.New(t)
	require.NoError(err)
	require.Equal(log.JobId, "4ba7c420-7444-44bc-a190-1bd4bad97b13")
	require.Equal(log.Status, "Invalid")
	require.Equal(log.StatusSummary, "Archive contains critical validation errors")
	require.Equal(len(log.TicketContents), 0)
	require.Equal(len(log.Issues), 3)
}

// testCmdLogValidSubmission mimicks an accepted submission.
func testCmdLogValidSubmission() int {
	fmt.Println(strings.TrimSpace(`
{
	"logFormatVersion": 1,
	"jobId": "3382aa04-e417-46a0-b1b4-42eebf85906c",
	"status": "Accepted",
	"statusSummary": "Ready for distribution",
	"statusCode": 0,
	"archiveFilename": "gon.zip",
	"uploadDate": "2019-11-06T00:51:10Z",
	"sha256": "1070be725b5b0c89b8dad699a9080a3bf5809fe68bfe8f84d6ff4a282d661fd1",
	"ticketContents": [
		{
		"path": "gon.zip/foo",
		"digestAlgorithm": "SHA-256",
		"cdhash": "b7049085e21423f102d6119bca93d57ebd903289",
		"arch": "x86_64"
		}
	],
	"issues": null
}
`))
	return 0
}

// testCmdLogInvalidSubmission mimicks an invalid submission.
func testCmdLogInvalidSubmission() int {
	fmt.Println(strings.TrimSpace(`
{
	"logFormatVersion": 1,
	"jobId": "4ba7c420-7444-44bc-a190-1bd4bad97b13",
	"status": "Invalid",
	"statusSummary": "Archive contains critical validation errors",
	"statusCode": 4000,
	"archiveFilename": "gon.zip",
	"uploadDate": "2019-11-06T00:54:22Z",
	"sha256": "c109f26d378fbf1efadc8987fdab79d2ce63155e8941823d4d11a907152e11a5",
	"ticketContents": null,
	"issues": [
		{
		"severity": "error",
		"code": null,
		"path": "gon.zip/foo",
		"message": "The binary is not signed.",
		"docUrl": null,
		"architecture": "x86_64"
		},
		{
		"severity": "error",
		"code": null,
		"path": "gon.zip/foo",
		"message": "The signature does not include a secure timestamp.",
		"docUrl": null,
		"architecture": "x86_64"
		},
		{
		"severity": "error",
		"code": null,
		"path": "gon.zip/foo",
		"message": "The executable does not have the hardened runtime enabled.",
		"docUrl": null,
		"architecture": "x86_64"
		}
	]
}	
`))
	return 0
}
