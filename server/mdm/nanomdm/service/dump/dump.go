// Pacakge dump is a NanoMDM service that dumps raw responses
package dump

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/service"
)

// Dumper is a service middleware that dumps MDM requests and responses
// to a file handle.
type Dumper struct {
	next service.CheckinAndCommandService
	file *os.File
	cmd  bool
	bst  bool
	usr  bool
	dm   bool
}

// New creates a new dumper service middleware.
func New(next service.CheckinAndCommandService, file *os.File) *Dumper {
	return &Dumper{
		next: next,
		file: file,
		cmd:  true,
		bst:  true,
		usr:  true,
		dm:   true,
	}
}

func (svc *Dumper) Authenticate(r *mdm.Request, m *mdm.Authenticate) error {
	_, _ = svc.file.Write(m.Raw)
	return svc.next.Authenticate(r, m)
}

func (svc *Dumper) TokenUpdate(r *mdm.Request, m *mdm.TokenUpdate) error {
	_, _ = svc.file.Write(m.Raw)
	return svc.next.TokenUpdate(r, m)
}

func (svc *Dumper) CheckOut(r *mdm.Request, m *mdm.CheckOut) error {
	_, _ = svc.file.Write(m.Raw)
	return svc.next.CheckOut(r, m)
}

func (svc *Dumper) UserAuthenticate(r *mdm.Request, m *mdm.UserAuthenticate) ([]byte, error) {
	_, _ = svc.file.Write(m.Raw)
	respBytes, err := svc.next.UserAuthenticate(r, m)
	if svc.usr && respBytes != nil && len(respBytes) > 0 {
		_, _ = svc.file.Write(respBytes)
	}
	return respBytes, err
}

func (svc *Dumper) SetBootstrapToken(r *mdm.Request, m *mdm.SetBootstrapToken) error {
	_, _ = svc.file.Write(m.Raw)
	return svc.next.SetBootstrapToken(r, m)
}

func (svc *Dumper) GetBootstrapToken(r *mdm.Request, m *mdm.GetBootstrapToken) (*mdm.BootstrapToken, error) {
	_, _ = svc.file.Write(m.Raw)
	bsToken, err := svc.next.GetBootstrapToken(r, m)
	if svc.bst && bsToken != nil && len(bsToken.BootstrapToken) > 0 {
		_, _ = svc.file.Write([]byte(fmt.Sprintf("Bootstrap token: %s\n", bsToken.BootstrapToken.String())))
	}
	return bsToken, err
}

func (svc *Dumper) GetToken(r *mdm.Request, m *mdm.GetToken) (*mdm.GetTokenResponse, error) {
	svc.file.Write(m.Raw) // nolint:errcheck
	token, err := svc.next.GetToken(r, m)
	if token != nil && len(token.TokenData) > 0 {
		b64 := base64.StdEncoding.EncodeToString(token.TokenData)
		svc.file.WriteString("GetToken TokenData: " + b64 + "\n") // nolint:errcheck
	}
	return token, err
}

func (svc *Dumper) CommandAndReportResults(r *mdm.Request, results *mdm.CommandResults) (*mdm.Command, error) {
	_, _ = svc.file.Write(results.Raw)
	cmd, err := svc.next.CommandAndReportResults(r, results)
	if svc.cmd && err != nil && cmd != nil && cmd.Raw != nil {
		_, _ = svc.file.Write(cmd.Raw)
	}
	return cmd, err
}

func (svc *Dumper) DeclarativeManagement(r *mdm.Request, m *mdm.DeclarativeManagement) ([]byte, error) {
	_, _ = svc.file.Write(m.Raw)
	if len(m.Data) > 0 {
		_, _ = svc.file.Write(m.Data)
	}
	respBytes, err := svc.next.DeclarativeManagement(r, m)
	if svc.dm && err != nil {
		_, _ = svc.file.Write(respBytes)
	}
	return respBytes, err
}
