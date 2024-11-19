package enrollment

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/test"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/test/protocol"
	"github.com/groob/plist"
)

var ErrAlreadyEnrolled = errors.New("already enrolled")

type Transport interface {
	// DoCheckIn performs an HTTP MDM check-in to the CheckInURL (or ServerURL).
	// The caller is responsible for closing the response body.
	DoCheckIn(context.Context, io.Reader) (*http.Response, error)

	// DoReportResultsAndFetchNext sends an HTTP MDM report-results-and-retrieve-next-command request to the ServerURL.
	// The caller is responsible for closing the response body.
	DoReportResultsAndFetchNext(ctx context.Context, report io.Reader) (*http.Response, error)
}

// Enrollment emulates an MDM enrollment.
// Currently it mostly emulates device channel enrollments.
type Enrollment struct {
	enrollID   mdm.EnrollID
	enrollment mdm.Enrollment
	push       mdm.Push

	cert *x509.Certificate
	key  crypto.PrivateKey

	serialNumber string
	unlockToken  []byte

	transport Transport

	enrolled bool
	enrollM  sync.Mutex
}

func loadAuthTokUpd(authPath, tokUpdPath string) (*mdm.Authenticate, *mdm.TokenUpdate, error) {
	authBytes, err := os.ReadFile(authPath)
	if err != nil {
		return nil, nil, err
	}
	msg, err := mdm.DecodeCheckin(authBytes)
	if err != nil {
		return nil, nil, err
	}
	auth, ok := msg.(*mdm.Authenticate)
	if !ok {
		return auth, nil, errors.New("not an Authenticate message")
	}
	tokUpdBytes, err := os.ReadFile(tokUpdPath)
	if err != nil {
		return auth, nil, err
	}
	msg, err = mdm.DecodeCheckin(tokUpdBytes)
	if err != nil {
		return auth, nil, err
	}
	tokUpd, ok := msg.(*mdm.TokenUpdate)
	if !ok {
		return auth, tokUpd, errors.New("not a TokenUpdate message")
	}
	return auth, tokUpd, nil
}

// NewFromCheckins loads device information from authenticate and tokenupdate files on disk.
func NewFromCheckins(doer protocol.Doer, serverURL, checkInURL, authenticatePath, tokenUpdatePath string) (*Enrollment, error) {
	auth, tokUpd, err := loadAuthTokUpd(authenticatePath, tokenUpdatePath)
	if err != nil {
		return nil, err
	}

	e := &Enrollment{
		enrollment:   auth.Enrollment,
		push:         tokUpd.Push,
		serialNumber: auth.SerialNumber,

		// we're assuming the IDs here are devices
		enrollID: mdm.EnrollID{Type: mdm.Device, ID: auth.UDID},
	}
	e.key, e.cert, err = test.SimpleSelfSignedRSAKeypair("TESTDEVICE", 2)

	e.transport = protocol.NewTransport(
		protocol.WithSignMessage(),
		protocol.WithIdentityProvider(e.GetIdentity),
		protocol.WithMDMURLs(serverURL, checkInURL),
		protocol.WithClient(doer),
	)

	return e, err
}

// NewRandomDeviceEnrollment creates a new randomly identified MDM enrollment.
func NewRandomDeviceEnrollment(doer protocol.Doer, topic, serverURL, checkInURL string) (*Enrollment, error) {
	udid := randString(32)
	e := &Enrollment{
		enrollment: mdm.Enrollment{UDID: udid},
		push: mdm.Push{
			Topic:     topic,
			PushMagic: randString(32),
			// Token:     []byte(randString(32)), // Token is populated in DoTokenUpdate()
		},
		serialNumber: randString(8),
		// unlockToken: ,
		enrollID: mdm.EnrollID{Type: mdm.Device, ID: udid},
	}
	var err error
	e.key, e.cert, err = test.SimpleSelfSignedRSAKeypair("TESTDEVICE", 2)

	e.transport = protocol.NewTransport(
		protocol.WithSignMessage(),
		protocol.WithIdentityProvider(e.GetIdentity),
		protocol.WithMDMURLs(serverURL, checkInURL),
		protocol.WithClient(doer),
	)

	return e, err
}

// GetIdentity supplies the identity certificate and key of this enrollment.
func (e *Enrollment) GetIdentity(context.Context) (*x509.Certificate, crypto.PrivateKey, error) {
	return e.cert, e.key, nil
}

// genAuthenticate creates an XML Plist Authenticate check-in message.
func (e *Enrollment) genAuthenticate() (io.Reader, error) {
	a := &mdm.Authenticate{
		Enrollment:   e.enrollment,
		MessageType:  mdm.MessageType{MessageType: "Authenticate"},
		Topic:        e.push.Topic,
		SerialNumber: e.serialNumber,
	}
	return test.PlistReader(a)
}

// genTokenUpdate creates an XML Plist TokenUpdate check-in message.
func (e *Enrollment) genTokenUpdate() (io.Reader, error) {
	t := &mdm.TokenUpdate{
		Enrollment:  e.enrollment,
		MessageType: mdm.MessageType{MessageType: "TokenUpdate"},
		Push:        e.push,
		UnlockToken: e.unlockToken,
	}
	return test.PlistReader(t)
}

// DoTokenUpdate sends a TokenUpdate to the MDM server.
func (e *Enrollment) DoTokenUpdate(ctx context.Context) error {
	e.enrollM.Lock()
	defer e.enrollM.Unlock()
	return e.doTokenUpdate(ctx)
}

// doTokenUpdate sends a TokenUpdate to the MDM server.
func (e *Enrollment) doTokenUpdate(ctx context.Context) error {
	// generate new random push token.
	// the token comes from Apple's APNs service. so we'll simulate this
	// by re-generating the token every time we do a TokenUpdate.
	e.push.Token = []byte(randString(32))

	// generate TokenUpdate check-in message
	msg, err := e.genTokenUpdate()
	if err != nil {
		return err
	}

	// send it to the MDM server
	resp, err := e.transport.DoCheckIn(ctx, msg)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// check for errors
	return HTTPErrors(resp)
}

// DoEnroll enrolls (or re-enrolls) this enrollment into MDM.
// Authenticate and TokenUpdate check-in messages are sent via the
// transport to the MDM server.
func (e *Enrollment) DoEnroll(ctx context.Context) error {
	e.enrollM.Lock()
	defer e.enrollM.Unlock()

	if e.enrolled {
		e.enrolled = false
	}

	// generate Authenticate check-in message
	auth, err := e.genAuthenticate()
	if err != nil {
		return err
	}

	// send it to the MDM server
	authResp, err := e.transport.DoCheckIn(ctx, auth)
	if err != nil {
		return err
	}

	// check for any errors
	if err = HTTPErrors(authResp); err != nil {
		authResp.Body.Close()
		return fmt.Errorf("enrollment authenticate check-in: %w", err)
	}
	authResp.Body.Close()

	err = e.doTokenUpdate(ctx)
	if err != nil {
		return err
	}

	e.enrolled = true

	return nil
}

// GetEnrollment returns the enrollment identifier data.
func (e *Enrollment) GetEnrollment() *mdm.Enrollment {
	return &e.enrollment
}

// ID returns the NanoMDM "normalized" enrollment ID.
func (e *Enrollment) ID() string {
	// we know we're only dealing with device IDs at this point.
	// make that assumption of the UDID for the normalized ID.
	return e.enrollment.UDID
}

// EnrollID returns the NanoMDM enroll ID.
func (e *Enrollment) EnrollID() *mdm.EnrollID {
	return &e.enrollID
}

func (e *Enrollment) NewMDMRequest(ctx context.Context) *mdm.Request {
	return &mdm.Request{
		Context:  ctx,
		EnrollID: e.EnrollID(),
	}
}

// GetPush returns the enrollment push info data.
func (e *Enrollment) GetPush() *mdm.Push {
	return &e.push
}

// DoReportAndFetch sends report to the MDM server.
// Any new command delivered will be in the response.
// The caller is responsible for closing the response body.
func (e *Enrollment) DoReportAndFetch(ctx context.Context, report io.Reader) (*http.Response, error) {
	return e.transport.DoReportResultsAndFetchNext(ctx, report)
}

// genSetBootstrapToken creates an XML Plist SetBootstrapToken check-in message.
func (e *Enrollment) genSetBootstrapToken(token []byte) (io.Reader, error) {
	msg := &mdm.SetBootstrapToken{
		Enrollment:     e.enrollment,
		MessageType:    mdm.MessageType{MessageType: "SetBootstrapToken"},
		BootstrapToken: mdm.BootstrapToken{BootstrapToken: make([]byte, base64.StdEncoding.EncodedLen(len(token)))},
	}
	base64.StdEncoding.Encode(msg.BootstrapToken.BootstrapToken, token)
	return test.PlistReader(msg)
}

// DoEscrowBootstrapToken sends the Bootstrap Token to the MDM server.
func (e *Enrollment) DoEscrowBootstrapToken(ctx context.Context, token []byte) error {
	r, err := e.genSetBootstrapToken(token)
	if err != nil {
		return err
	}

	// send it to the MDM server
	resp, err := e.transport.DoCheckIn(ctx, r)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// check for errors
	return HTTPErrors(resp)
}

// genGetBootstrapToken creates an XML Plist GetBootstrapToken check-in message.
func (e *Enrollment) genGetBootstrapToken() (io.Reader, error) {
	msg := &mdm.GetBootstrapToken{
		Enrollment:  e.enrollment,
		MessageType: mdm.MessageType{MessageType: "GetBootstrapToken"},
	}
	return test.PlistReader(msg)
}

// DoGetBootstrapToken retrieves the Bootstrap Token from the MDM erver.
func (e *Enrollment) DoGetBootstrapToken(ctx context.Context) (*mdm.BootstrapToken, error) {
	r, err := e.genGetBootstrapToken()
	if err != nil {
		return nil, err
	}

	// send it to the MDM server
	resp, err := e.transport.DoCheckIn(ctx, r)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, Limit10KiB))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, NewHTTPError(resp, body)
	}

	var tok *mdm.BootstrapToken
	if len(body) > 0 {
		tok = new(mdm.BootstrapToken)
		err = plist.Unmarshal(body, tok)
	}
	return tok, err
}

func randString(n int) string {
	b := make([]byte, n)
	rand.Read(b) // nolint:errcheck
	return fmt.Sprintf("%x", b)
}
