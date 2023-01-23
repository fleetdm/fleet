//go:build linux
// +build linux

package fscrypt_info

import (
	"fmt"
	"io"
	"log"

	"github.com/google/fscrypt/actions"
	"github.com/google/fscrypt/keyring"
	"github.com/google/fscrypt/metadata"
)

type Info struct {
	Encrypted      bool
	Locked         string
	Mountpoint     string
	FilesystemType string
	Device         string
	Path           string
	ContentsAlgo   string
	FilenameAlgo   string
}

func GetInfo(dirpath string) (*Info, error) {
	origLog := log.Writer()
	defer func() {
		log.SetOutput(origLog)
	}()
	log.SetOutput(io.Discard)

	fsctx, err := actions.NewContextFromPath(dirpath, nil)
	if err != nil {
		return nil, fmt.Errorf("new context: %w", err)
	}

	pol, err := actions.GetPolicyFromPath(fsctx, dirpath)
	switch err.(type) {
	case nil:
		break
	case *metadata.ErrNotEncrypted:
		return &Info{Path: dirpath, Encrypted: false}, nil
	default:
		return nil, fmt.Errorf("get policy for %s: %w", dirpath, err)
	}

	return &Info{
		Path:           dirpath,
		Locked:         policyUnlockedStatus(pol),
		Encrypted:      true,
		Mountpoint:     pol.Context.Mount.Path,
		FilesystemType: pol.Context.Mount.FilesystemType,
		Device:         pol.Context.Mount.Device,
		ContentsAlgo:   pol.Options().Contents.String(),
		FilenameAlgo:   pol.Options().Filenames.String(),
	}, nil

}

// policyUnlockedStatus is from
// https://github.com/google/fscrypt/blob/dad0c1158455dcfd9acbd219a04ef348bf454332/cmd/fscrypt/status.go#L67
func policyUnlockedStatus(policy *actions.Policy) string {
	status := policy.GetProvisioningStatus()

	switch status {
	case keyring.KeyPresent, keyring.KeyPresentButOnlyOtherUsers:
		return "no"
	case keyring.KeyAbsent:
		return "yes"
	case keyring.KeyAbsentButFilesBusy:
		return "partially (incompletely locked)"
	default:
		return "unknown"
	}
}
