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

// ReplaceIdentityRandom changes the certificate private key to a random certificate and key.
func ReplaceIdentityRandom(e *Enrollment) error {
	var err error
	e.key, e.cert, err = test.SimpleSelfSignedRSAKeypair("TESTDEVICE", 2)
	return err
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

// GenAuthenticate creates an XML Plist Authenticate check-in message.
func (e *Enrollment) GenAuthenticate() (io.Reader, error) {
	a := &mdm.Authenticate{
		Enrollment:   e.enrollment,
		MessageType:  mdm.MessageType{MessageType: "Authenticate"},
		Topic:        e.push.Topic,
		SerialNumber: e.serialNumber,
	}
	return test.PlistReader(a)
}

// GenTokenUpdate creates an XML Plist TokenUpdate check-in message.
func (e *Enrollment) GenTokenUpdate() (io.Reader, error) {
	t := &mdm.TokenUpdate{
		Enrollment:  e.enrollment,
		MessageType: mdm.MessageType{MessageType: "TokenUpdate"},
		Push:        e.push,
		UnlockToken: e.unlockToken,
	}
	return test.PlistReader(t)
}

// doAuthenticate sends an Authenticate check-in message to the MDM server.
func (e *Enrollment) doAuthenticate(ctx context.Context) error {
	e.enrolled = false

	// generate Authenticate check-in message
	auth, err := e.GenAuthenticate()
	if err != nil {
		return err
	}

	// send it to the MDM server
	authResp, err := e.transport.DoCheckIn(ctx, auth)
	if err != nil {
		return err
	}
	defer authResp.Body.Close()

	// check for any errors
	return HTTPErrors(authResp)
}

// DoAuthenticate sends an Authenticate check-in message to the MDM server.
func (e *Enrollment) DoAuthenticate(ctx context.Context) error {
	e.enrollM.Lock()
	defer e.enrollM.Unlock()
	return e.doAuthenticate(ctx)
}

// doTokenUpdate sends a TokenUpdate check-in message to the MDM server.
// A new random push token is generated for the device.
func (e *Enrollment) doTokenUpdate(ctx context.Context) error {
	// generate new random push token.
	// the token comes from Apple's APNs service. so we'll simulate this
	// by re-generating the token every time we do a TokenUpdate.
	e.push.Token = []byte(randString(32))

	// generate TokenUpdate check-in message
	msg, err := e.GenTokenUpdate()
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

// DoTokenUpdate sends a TokenUpdate check-in message to the MDM server.
// A new random push token is generated for the device.
func (e *Enrollment) DoTokenUpdate(ctx context.Context) error {
	e.enrollM.Lock()
	defer e.enrollM.Unlock()
	return e.doTokenUpdate(ctx)
}

// DoEnroll enrolls (or re-enrolls) this enrollment into MDM.
// Authenticate and TokenUpdate check-in messages are sent via the
// transport to the MDM server.
func (e *Enrollment) DoEnroll(ctx context.Context) error {
	e.enrollM.Lock()
	defer e.enrollM.Unlock()

	err := e.doAuthenticate(ctx)
	if err != nil {
		return fmt.Errorf("authenticate check-in: %w", err)
	}

	err = e.doTokenUpdate(ctx)
	if err != nil {
		return fmt.Errorf("tokenupdate check-in: %w", err)
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
		Context:     ctx,
		EnrollID:    e.EnrollID(),
		Certificate: e.cert,
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
	b64Token := base64.StdEncoding.EncodeToString(token)
	msg := &mdm.SetBootstrapToken{
		Enrollment:     e.enrollment,
		MessageType:    mdm.MessageType{MessageType: "SetBootstrapToken"},
		BootstrapToken: mdm.BootstrapToken{BootstrapToken: []byte(b64Token)},
	}
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
