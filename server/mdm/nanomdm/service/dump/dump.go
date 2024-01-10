// Pacakge dump is a NanoMDM service that dumps raw responses
package dump

import (
	"fmt"
	"os"

	"github.com/micromdm/nanomdm/mdm"
	"github.com/micromdm/nanomdm/service"
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
	svc.file.Write(m.Raw)
	return svc.next.Authenticate(r, m)
}

func (svc *Dumper) TokenUpdate(r *mdm.Request, m *mdm.TokenUpdate) error {
	svc.file.Write(m.Raw)
	return svc.next.TokenUpdate(r, m)
}

func (svc *Dumper) CheckOut(r *mdm.Request, m *mdm.CheckOut) error {
	svc.file.Write(m.Raw)
	return svc.next.CheckOut(r, m)
}

func (svc *Dumper) UserAuthenticate(r *mdm.Request, m *mdm.UserAuthenticate) ([]byte, error) {
	svc.file.Write(m.Raw)
	respBytes, err := svc.next.UserAuthenticate(r, m)
	if svc.usr && respBytes != nil && len(respBytes) > 0 {
		svc.file.Write(respBytes)
	}
	return respBytes, err
}

func (svc *Dumper) SetBootstrapToken(r *mdm.Request, m *mdm.SetBootstrapToken) error {
	svc.file.Write(m.Raw)
	return svc.next.SetBootstrapToken(r, m)
}

func (svc *Dumper) GetBootstrapToken(r *mdm.Request, m *mdm.GetBootstrapToken) (*mdm.BootstrapToken, error) {
	svc.file.Write(m.Raw)
	bsToken, err := svc.next.GetBootstrapToken(r, m)
	if svc.bst && bsToken != nil && len(bsToken.BootstrapToken) > 0 {
		svc.file.Write([]byte(fmt.Sprintf("Bootstrap token: %s\n", bsToken.BootstrapToken.String())))
	}
	return bsToken, err
}

func (svc *Dumper) CommandAndReportResults(r *mdm.Request, results *mdm.CommandResults) (*mdm.Command, error) {
	svc.file.Write(results.Raw)
	cmd, err := svc.next.CommandAndReportResults(r, results)
	if svc.cmd && err != nil && cmd != nil && cmd.Raw != nil {
		svc.file.Write(cmd.Raw)
	}
	return cmd, err
}

func (svc *Dumper) DeclarativeManagement(r *mdm.Request, m *mdm.DeclarativeManagement) ([]byte, error) {
	svc.file.Write(m.Raw)
	if len(m.Data) > 0 {
		svc.file.Write(m.Data)
	}
	respBytes, err := svc.next.DeclarativeManagement(r, m)
	if svc.dm && err != nil {
		svc.file.Write(respBytes)
	}
	return respBytes, err
}
