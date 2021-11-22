//+build windows

package platform

import (
	"fmt"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"

	"github.com/hectane/go-acl"
)

const (
	fullControl    = uint32(2032127)
	readAndExecute = uint32(131241)
)

// ChmodExecutableDirectory sets the appropriate permissions on the parent
// directory of an executable file. On Windows this involves setting the
// appropriate ACLs.
func ChmodExecutableDirectory(path string) error {
	if err := acl.Apply(
		path,
		true,
		false,
		acl.GrantSid(fullControl, constant.SystemSID),
		acl.GrantSid(fullControl, constant.AdminSID),
		acl.GrantSid(readAndExecute, constant.UserSID),
	); err != nil {
		return fmt.Errorf("apply ACLs: %w", err)
	}

	return nil
}

// ChmodExecutable sets the appropriate permissions on an executable file. On
// Windows this involves setting the appropriate ACLs.
func ChmodExecutable(path string) error {
	if err := acl.Apply(
		path,
		true,
		false,
		acl.GrantSid(fullControl, constant.SystemSID),
		acl.GrantSid(fullControl, constant.AdminSID),
		acl.GrantSid(readAndExecute, constant.UserSID),
	); err != nil {
		return fmt.Errorf("apply ACLs: %w", err)
	}

	return nil
}
