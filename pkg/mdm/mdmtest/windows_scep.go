package mdmtest

import (
	"context"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/xml"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/microsoft/syncml"
	scepclient "github.com/fleetdm/fleet/v4/server/mdm/scep/client"
	"github.com/fleetdm/fleet/v4/server/mdm/scep/x509util"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/google/uuid"
	smallstepscep "github.com/smallstep/scep"
)

// WindowsMDMSCEPAlertType is the OMA-DM Alert <Type> emitted by real Windows MDM clients to
// report the outcome of an asynchronous SCEP CertificateInstall to the MDM server. Fleet's
// processGenericAlert ignores it (only unenrollment alerts are acted on), so this is for
// parity with real-client traffic and for visibility in server logs.
const WindowsMDMSCEPAlertType = "com.microsoft:mdm.SCEP"

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
	HashAlg     string
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
// are handled. Incomplete CSPs (missing ServerURL or SubjectName) are dropped silently.
func (c *TestWindowsMDMClient) ExtractSCEPCommands(cmds map[string]fleet.ProtoCmdOperation) []SCEPCommand {
	byID := map[string]*SCEPCommand{}

	get := func(uniqueID string) *SCEPCommand {
		sc, ok := byID[uniqueID]
		if !ok {
			sc = &SCEPCommand{UniqueID: uniqueID, KeyLength: 2048, HashAlg: "SHA-256"}
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
		case "HashAlgorithm":
			if val != "" {
				sc.HashAlg = val
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
			atomicID := op.Cmd.CmdID.Value
			before := collectKeys(byID)
			for _, child := range op.Cmd.AddCommands {
				walkChild(child, fleet.CmdAdd)
			}
			for _, child := range op.Cmd.ExecCommands {
				walkChild(child, fleet.CmdExec)
			}
			// Stamp the Atomic CmdID onto every CSP first observed under this Atomic.
			for k, sc := range byID {
				if _, seen := before[k]; seen {
					continue
				}
				if sc.AtomicCmdID == "" {
					sc.AtomicCmdID = atomicID
				}
			}
		case fleet.CmdAdd:
			walkChild(op.Cmd, fleet.CmdAdd)
		case fleet.CmdExec:
			walkChild(op.Cmd, fleet.CmdExec)
		}
	}

	out := make([]SCEPCommand, 0, len(byID))
	for _, sc := range byID {
		if sc.ServerURL == "" || sc.SubjectName == "" {
			// Not a complete SCEP install request - skip.
			continue
		}
		out = append(out, *sc)
	}
	return out
}

func collectKeys(m map[string]*SCEPCommand) map[string]struct{} {
	out := make(map[string]struct{}, len(m))
	for k := range m {
		out[k] = struct{}{}
	}
	return out
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

// AppendSCEPInstallResponses appends Status acks for each SCEP CSP it finds, kicks off the SCEP
// exchanges in goroutines, and queues a generic Alert per CSP to be flushed on the next sync
// once the exchange completes.
//
// Returned values:
//   - handled: CmdIDs that the helper has already ACKed. Callers iterating remaining commands
//     should skip these so they don't double-ACK.
//   - done: receives one SCEPResult per CSP and is closed when all exchanges complete. Callers
//     that don't need synchronization can ignore it.
func (c *TestWindowsMDMClient) AppendSCEPInstallResponses(
	ctx context.Context,
	cmds map[string]fleet.ProtoCmdOperation,
	msgID string,
	logger *slog.Logger,
) (handled map[string]struct{}, done <-chan SCEPResult) {
	handled = map[string]struct{}{}
	sceps := c.ExtractSCEPCommands(cmds)
	if len(sceps) == 0 {
		ch := make(chan SCEPResult)
		close(ch)
		return handled, ch
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

	out := make(chan SCEPResult, len(sceps))
	var wg sync.WaitGroup
	for _, sc := range sceps {
		wg.Add(1)
		go func(sc SCEPCommand) {
			defer wg.Done()
			cert, err := c.RunSCEP(ctx, sc, logger)
			c.queueSCEPAlert(sc, err)
			out <- SCEPResult{UniqueID: sc.UniqueID, Cert: cert, Err: err}
		}(sc)
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	return handled, out
}

// RunSCEP performs a single SCEP exchange against sc.ServerURL using the challenge and subject
// from the CSP, and returns the issued certificate. It is exposed separately from
// AppendSCEPInstallResponses so callers (and tests) can drive the exchange synchronously when
// they want.
func (c *TestWindowsMDMClient) RunSCEP(ctx context.Context, sc SCEPCommand, logger *slog.Logger) (*x509.Certificate, error) {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	if sc.ServerURL == "" {
		return nil, errors.New("scep command missing ServerURL")
	}

	timeout := 30 * time.Second
	client, err := scepclient.New(sc.ServerURL, logger,
		scepclient.WithTimeout(&timeout),
		scepclient.Insecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("creating scep client: %w", err)
	}

	caResp, _, err := client.GetCACert(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("scep get ca cert: %w", err)
	}
	caCerts, err := x509.ParseCertificates(caResp)
	if err != nil {
		return nil, fmt.Errorf("scep parse ca certs: %w", err)
	}
	if len(caCerts) == 0 {
		return nil, errors.New("scep get ca cert returned no certificates")
	}

	keyBits := sc.KeyLength
	if keyBits <= 0 {
		keyBits = 2048
	}
	privKey, err := rsa.GenerateKey(cryptorand.Reader, keyBits)
	if err != nil {
		return nil, fmt.Errorf("scep generate rsa key: %w", err)
	}
	subject, err := parseSCEPSubject(sc.SubjectName)
	if err != nil {
		return nil, fmt.Errorf("scep parse subject: %w", err)
	}

	csrTpl := x509util.CertificateRequest{
		CertificateRequest: x509.CertificateRequest{
			Subject:            subject,
			SignatureAlgorithm: signatureAlgorithmForHash(sc.HashAlg),
		},
		ChallengePassword: sc.Challenge,
	}
	csrDER, err := x509util.CreateCertificateRequest(cryptorand.Reader, &csrTpl, privKey)
	if err != nil {
		return nil, fmt.Errorf("scep create csr: %w", err)
	}
	csr, err := x509.ParseCertificateRequest(csrDER)
	if err != nil {
		return nil, fmt.Errorf("scep parse csr: %w", err)
	}

	// SCEP requires the request to be signed by an existing cert. For first-time enrollment we
	// use a short-lived self-signed cert wrapping the same key as the CSR.
	signerCert, err := selfSignedSignerCert(privKey, subject)
	if err != nil {
		return nil, fmt.Errorf("scep create self-signed signer: %w", err)
	}

	pkiReq := &smallstepscep.PKIMessage{
		MessageType: smallstepscep.PKCSReq,
		Recipients:  caCerts,
		SignerKey:   privKey,
		SignerCert:  signerCert,
	}
	msg, err := smallstepscep.NewCSRRequest(csr, pkiReq)
	if err != nil {
		return nil, fmt.Errorf("scep new csr request: %w", err)
	}

	respBytes, err := client.PKIOperation(ctx, msg.Raw)
	if err != nil {
		return nil, fmt.Errorf("scep pki operation: %w", err)
	}
	pkiResp, err := smallstepscep.ParsePKIMessage(respBytes, smallstepscep.WithCACerts(msg.Recipients))
	if err != nil {
		return nil, fmt.Errorf("scep parse pki response: %w", err)
	}
	if pkiResp.PKIStatus != smallstepscep.SUCCESS {
		return nil, fmt.Errorf("scep pki status %v (failInfo=%v)", pkiResp.PKIStatus, pkiResp.FailInfo)
	}
	if err := pkiResp.DecryptPKIEnvelope(signerCert, privKey); err != nil {
		return nil, fmt.Errorf("scep decrypt pki envelope: %w", err)
	}
	if pkiResp.CertRepMessage == nil || pkiResp.CertRepMessage.Certificate == nil {
		return nil, errors.New("scep response contained no certificate")
	}
	return pkiResp.CertRepMessage.Certificate, nil
}

// queueSCEPAlert builds a generic Alert reporting the SCEP outcome and stages it on the test
// client. The alert is flushed on the next SendResponse.
func (c *TestWindowsMDMClient) queueSCEPAlert(sc SCEPCommand, scepErr error) {
	status := "success"
	if scepErr != nil {
		status = "failure"
	}
	typeContent := WindowsMDMSCEPAlertType
	formatContent := "chr"
	data := fmt.Sprintf(`{"unique_id":%q,"status":%q}`, sc.UniqueID, status)

	alert := fleet.SyncMLCmd{
		XMLName: xml.Name{Local: fleet.CmdAlert},
		CmdID:   fleet.CmdID{Value: uuid.NewString()},
		Data:    ptr.String(syncml.CmdAlertGeneric),
		Items: []fleet.CmdItem{{
			Meta: &fleet.Meta{
				Type:   &fleet.MetaAttr{XMLNS: syncml.SyncMLMetaNamespace, Content: ptr.String(typeContent)},
				Format: &fleet.MetaAttr{XMLNS: syncml.SyncMLMetaNamespace, Content: ptr.String(formatContent)},
			},
			Data: &fleet.RawXmlData{Content: data},
		}},
	}

	c.pendingAlertsMu.Lock()
	c.pendingAlerts = append(c.pendingAlerts, alert)
	c.pendingAlertsMu.Unlock()
}

// takePendingAlerts removes and returns any alerts queued by async helpers.
func (c *TestWindowsMDMClient) takePendingAlerts() []fleet.SyncMLCmd {
	c.pendingAlertsMu.Lock()
	defer c.pendingAlertsMu.Unlock()
	if len(c.pendingAlerts) == 0 {
		return nil
	}
	out := c.pendingAlerts
	c.pendingAlerts = nil
	return out
}

func newStatusCmd(msgID, cmdRef, cmd, status string) fleet.SyncMLCmd {
	return fleet.SyncMLCmd{
		XMLName: xml.Name{Local: fleet.CmdStatus},
		MsgRef:  &msgID,
		CmdRef:  &cmdRef,
		Cmd:     ptr.String(cmd),
		Data:    ptr.String(status),
		CmdID:   fleet.CmdID{Value: uuid.NewString()},
	}
}

// parseSCEPSubject parses a SubjectName CSP value of the form CN=foo,OU=bar into a pkix.Name.
// Only the simple shapes Fleet sends are supported (CN, OU, O, C, L, ST). Escaped commas in
// values are not supported because the CSP doesn't deliver escapes either.
func parseSCEPSubject(s string) (pkix.Name, error) {
	n := pkix.Name{}
	if strings.TrimSpace(s) == "" {
		return n, errors.New("empty subject")
	}
	for part := range strings.SplitSeq(s, ",") {
		left, right, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		key := strings.ToUpper(strings.TrimSpace(left))
		val := strings.TrimSpace(right)
		switch key {
		case "CN":
			n.CommonName = val
		case "OU":
			n.OrganizationalUnit = append(n.OrganizationalUnit, val)
		case "O":
			n.Organization = append(n.Organization, val)
		case "C":
			n.Country = append(n.Country, val)
		case "L":
			n.Locality = append(n.Locality, val)
		case "ST":
			n.Province = append(n.Province, val)
		}
	}
	if n.CommonName == "" && len(n.OrganizationalUnit) == 0 {
		return n, fmt.Errorf("subject %q has no recognized RDN", s)
	}
	return n, nil
}

func signatureAlgorithmForHash(hashAlg string) x509.SignatureAlgorithm {
	switch strings.ToUpper(strings.TrimSpace(hashAlg)) {
	case "SHA-384", "SHA384":
		return x509.SHA384WithRSA
	case "SHA-512", "SHA512":
		return x509.SHA512WithRSA
	default:
		// smallstep's x509util.addChallenge only knows SHA-256/384/512 with RSA, so SHA-1 is silently
		// upgraded to SHA-256. Real Windows clients ignore the HashAlgorithm node when the requested
		// hash is unavailable too. Empty/unrecognized values also land here.
		return x509.SHA256WithRSA
	}
}

func selfSignedSignerCert(key *rsa.PrivateKey, subject pkix.Name) (*x509.Certificate, error) {
	now := time.Now()
	tpl := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               subject,
		NotBefore:             now.Add(-1 * time.Minute),
		NotAfter:              now.Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(cryptorand.Reader, &tpl, &tpl, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(der)
}
