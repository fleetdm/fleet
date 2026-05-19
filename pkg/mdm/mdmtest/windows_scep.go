package mdmtest

import (
	"cmp"
	"context"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/xml"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/microsoft/syncml"
	"github.com/google/uuid"
)

// SCEPCommand describes a single SCEP CertificateInstall CSP that the test client received.
// Returned by ExtractSCEPCommands. All fields except UniqueID may be empty if the corresponding
// CSP node was not present.
type SCEPCommand struct {
	UniqueID    string // path segment between /SCEP/ and /Install/
	AtomicCmdID string // CmdID of the wrapping <Atomic>, if any
	EnrollCmdID string // CmdID of the <Exec> on /Install/Enroll
	AddCmdIDs   []string

	ServerURL   string
	Challenge   string
	SubjectName string
	KeyLength   int
}

// SCEPResult is emitted on the channel returned by AppendSCEPInstallResponses once the async
// SCEP exchange for a given CSP completes.
type SCEPResult struct {
	UniqueID string
	Cert     *x509.Certificate
	Err      error
}

// ExtractSCEPCommands walks the commands map and groups SCEP CSP nodes by UniqueID. Both
// ./Device/... and ./User/... LocURIs are recognized. Both Atomic-wrapped and flat layouts
// are handled.
//
// Returns two slices: complete CSPs that have ServerURL, SubjectName, and the /Install/Enroll
// Exec (ready to drive a SCEP exchange); and incomplete CSPs that are missing at least one
// of those. Incomplete CSPs are surfaced rather than dropped so callers can fail loudly on a
// malformed profile or parser bug instead of silently never running SCEP.
func (c *TestWindowsMDMClient) ExtractSCEPCommands(cmds map[string]fleet.ProtoCmdOperation) (complete, incomplete []SCEPCommand) {
	byID := map[string]*SCEPCommand{}
	// currentAtomicID is set while walking the children of an Atomic op so that any new SCEP
	// entry created during that walk gets the Atomic's CmdID stamped on it at creation time.
	// Entries created outside an Atomic (standalone Add/Exec) get an empty AtomicCmdID.
	var currentAtomicID string

	get := func(uniqueID string) *SCEPCommand {
		sc, ok := byID[uniqueID]
		if !ok {
			sc = &SCEPCommand{UniqueID: uniqueID, KeyLength: 2048, AtomicCmdID: currentAtomicID}
			byID[uniqueID] = sc
		}
		return sc
	}

	addNode := func(cmdID, target, data string) {
		path, ok := scepInstallPath(target)
		if !ok {
			return
		}
		sc := get(path.UniqueID)
		sc.AddCmdIDs = append(sc.AddCmdIDs, cmdID)
		val := strings.TrimSpace(data)
		switch path.Field {
		case "ServerURL":
			sc.ServerURL = val
		case "Challenge":
			sc.Challenge = val
		case "SubjectName":
			sc.SubjectName = val
		case "KeyLength":
			if n, err := strconv.Atoi(val); err == nil && n > 0 {
				sc.KeyLength = n
			}
		}
	}

	addExec := func(cmdID, target string) {
		path, ok := scepInstallPath(target)
		if !ok || path.Field != "Enroll" {
			return
		}
		get(path.UniqueID).EnrollCmdID = cmdID
	}

	walkChild := func(child fleet.SyncMLCmd, verb string) {
		target := child.GetTargetURI()
		data := child.GetTargetData()
		switch verb {
		case fleet.CmdAdd:
			addNode(child.CmdID.Value, target, data)
		case fleet.CmdExec:
			addExec(child.CmdID.Value, target)
		}
	}

	for _, op := range cmds {
		switch op.Verb {
		case fleet.CmdAtomic:
			currentAtomicID = op.Cmd.CmdID.Value
			for _, child := range op.Cmd.AddCommands {
				walkChild(child, fleet.CmdAdd)
			}
			for _, child := range op.Cmd.ExecCommands {
				walkChild(child, fleet.CmdExec)
			}
			currentAtomicID = ""
		case fleet.CmdAdd:
			walkChild(op.Cmd, fleet.CmdAdd)
		case fleet.CmdExec:
			walkChild(op.Cmd, fleet.CmdExec)
		}
	}

	for _, sc := range byID {
		if sc.ServerURL == "" || sc.SubjectName == "" || sc.EnrollCmdID == "" {
			incomplete = append(incomplete, *sc)
			continue
		}
		complete = append(complete, *sc)
	}
	// Map iteration order is nondeterministic; sort both slices by UniqueID so tests asserting
	// on multi-CSP profiles aren't flaky.
	byUniqueID := func(a, b SCEPCommand) int { return cmp.Compare(a.UniqueID, b.UniqueID) }
	slices.SortFunc(complete, byUniqueID)
	slices.SortFunc(incomplete, byUniqueID)
	return complete, incomplete
}

// missingSCEPFields returns a comma-separated list of the required fields that are not set on
// sc. Used to build a descriptive error for incomplete CSPs surfaced by ExtractSCEPCommands.
func missingSCEPFields(sc SCEPCommand) string {
	var missing []string
	if sc.ServerURL == "" {
		missing = append(missing, "ServerURL")
	}
	if sc.SubjectName == "" {
		missing = append(missing, "SubjectName")
	}
	if sc.EnrollCmdID == "" {
		missing = append(missing, "/Install/Enroll Exec")
	}
	return strings.Join(missing, ", ")
}

type scepCSPPath struct {
	UniqueID string
	Field    string
}

// scepInstallPath parses LocURIs of the form
//
//	./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/<UniqueID>/Install/<Field>
//	./User/Vendor/MSFT/ClientCertificateInstall/SCEP/<UniqueID>/Install/<Field>
//
// returning the unique ID and field name. Whitespace and newlines that may be present in the
// LocURI text node (test fixtures often include indentation) are stripped. Returns ok=false
// for anything else.
func scepInstallPath(loc string) (scepCSPPath, bool) {
	s := strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == '\t' || r == ' ' {
			return -1
		}
		return r
	}, loc)
	const marker = "/Vendor/MSFT/ClientCertificateInstall/SCEP/"
	_, rest, ok := strings.Cut(s, marker)
	if !ok {
		return scepCSPPath{}, false
	}
	left, right, ok := strings.Cut(rest, "/Install/")
	if !ok {
		return scepCSPPath{}, false
	}
	uniqueID := strings.Trim(left, "/")
	field := strings.Trim(right, "/")
	if uniqueID == "" || field == "" {
		return scepCSPPath{}, false
	}
	return scepCSPPath{UniqueID: uniqueID, Field: field}, true
}

// AppendSCEPInstallResponses appends Status acks for each complete SCEP CSP it finds and kicks
// off the SCEP exchanges in goroutines.
//
// Incomplete CSPs (missing ServerURL, SubjectName, or the /Install/Enroll Exec) are not ACKed
// here: the caller's normal iteration over cmds will ACK whatever partial Add/Atomic CmdIDs
// exist with its usual status. Each incomplete CSP is emitted on the result channel as a
// SCEPResult with a descriptive Err so the caller fails loudly instead of silently never
// running SCEP.
//
// Returned values:
//   - handled: CmdIDs that the helper has already ACKed. Callers iterating remaining commands
//     should skip these so they don't double-ACK.
//   - done: receives one SCEPResult per complete-or-incomplete CSP and is closed when all
//     exchanges complete. Callers that don't need synchronization can ignore it.
//   - hasWork: true if any SCEP CSPs (complete or incomplete) were found. When false, done is
//     a closed empty channel and there's no reason for the caller to spawn a drain goroutine.
func (c *TestWindowsMDMClient) AppendSCEPInstallResponses(
	ctx context.Context,
	cmds map[string]fleet.ProtoCmdOperation,
	msgID string,
	logger *slog.Logger,
) (handled map[string]struct{}, done <-chan SCEPResult, hasWork bool) {
	handled = map[string]struct{}{}
	sceps, incomplete := c.ExtractSCEPCommands(cmds)
	if len(sceps) == 0 && len(incomplete) == 0 {
		ch := make(chan SCEPResult)
		close(ch)
		return handled, ch, false
	}
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}

	ackedAtomic := map[string]struct{}{}
	for _, sc := range sceps {
		if sc.AtomicCmdID != "" {
			if _, ok := ackedAtomic[sc.AtomicCmdID]; !ok {
				c.AppendResponse(newStatusCmd(msgID, sc.AtomicCmdID, fleet.CmdAtomic, syncml.CmdStatusOK))
				ackedAtomic[sc.AtomicCmdID] = struct{}{}
			}
			handled[sc.AtomicCmdID] = struct{}{}
			continue
		}
		// Flat layout: ACK each Add and the Exec individually.
		for _, id := range sc.AddCmdIDs {
			c.AppendResponse(newStatusCmd(msgID, id, fleet.CmdAdd, syncml.CmdStatusOK))
			handled[id] = struct{}{}
		}
		if sc.EnrollCmdID != "" {
			c.AppendResponse(newStatusCmd(msgID, sc.EnrollCmdID, fleet.CmdExec, syncml.CmdStatusOK))
			handled[sc.EnrollCmdID] = struct{}{}
		}
	}

	out := make(chan SCEPResult, len(sceps)+len(incomplete))
	for _, sc := range incomplete {
		out <- SCEPResult{
			UniqueID: sc.UniqueID,
			Err:      fmt.Errorf("incomplete SCEP CSP for %q: missing %s", sc.UniqueID, missingSCEPFields(sc)),
		}
	}
	var wg sync.WaitGroup
	for _, sc := range sceps {
		wg.Add(1)
		go func(sc SCEPCommand) {
			defer wg.Done()
			cert, err := c.RunSCEP(ctx, sc, logger)
			out <- SCEPResult{UniqueID: sc.UniqueID, Cert: cert, Err: err}
		}(sc)
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	return handled, out, true
}

// RunSCEP performs a single SCEP exchange against sc.ServerURL using the challenge and subject
// from the CSP, and returns the issued certificate. It is exposed separately from
// AppendSCEPInstallResponses so callers (and tests) can drive the exchange synchronously when
// they want.
func (c *TestWindowsMDMClient) RunSCEP(ctx context.Context, sc SCEPCommand, logger *slog.Logger) (*x509.Certificate, error) {
	subject, err := parseSCEPSubject(sc.SubjectName)
	if err != nil {
		return nil, fmt.Errorf("scep parse subject: %w", err)
	}
	cert, _, err := performSCEPExchange(ctx, scepExchangeRequest{
		URL:       sc.ServerURL,
		Subject:   subject,
		Challenge: sc.Challenge,
		KeyBits:   sc.KeyLength,
	}, logger)
	return cert, err
}

func newStatusCmd(msgID, cmdRef, cmd, status string) fleet.SyncMLCmd {
	return fleet.SyncMLCmd{
		XMLName: xml.Name{Local: fleet.CmdStatus},
		MsgRef:  &msgID,
		CmdRef:  &cmdRef,
		Cmd:     &cmd,
		Data:    &status,
		CmdID:   fleet.CmdID{Value: uuid.NewString()},
	}
}

// parseSCEPSubject parses a SubjectName CSP value of the form CN=foo,OU=bar into a pkix.Name.
// Only CN and OU are recognized. Escaped commas in values are not supported.
func parseSCEPSubject(s string) (pkix.Name, error) {
	n := pkix.Name{}
	if strings.TrimSpace(s) == "" {
		return n, errors.New("empty subject")
	}
	recognized := false
	for part := range strings.SplitSeq(s, ",") {
		left, right, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		val := strings.TrimSpace(right)
		switch strings.ToUpper(strings.TrimSpace(left)) {
		case "CN":
			n.CommonName = val
			recognized = true
		case "OU":
			n.OrganizationalUnit = append(n.OrganizationalUnit, val)
			recognized = true
		}
	}
	if !recognized {
		return n, fmt.Errorf("subject %q has no recognized RDN (expected CN or OU)", s)
	}
	return n, nil
}
