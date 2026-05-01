package fleet

import (
	"context"
	"io"
)

const OrgLogoMaxFileSize = 100 * 1024

type OrgLogoMode string

const (
	OrgLogoModeLight OrgLogoMode = "light"
	OrgLogoModeDark  OrgLogoMode = "dark"
	OrgLogoModeAll   OrgLogoMode = "all"
)

func (m OrgLogoMode) IsValid() bool {
	return m == OrgLogoModeLight || m == OrgLogoModeDark || m == OrgLogoModeAll
}

func (m OrgLogoMode) IsStorable() bool {
	return m == OrgLogoModeLight || m == OrgLogoModeDark
}

func (m OrgLogoMode) Modes() []OrgLogoMode {
	switch m {
	case OrgLogoModeAll:
		return []OrgLogoMode{OrgLogoModeLight, OrgLogoModeDark}
	case OrgLogoModeLight, OrgLogoModeDark:
		return []OrgLogoMode{m}
	}
	return nil
}

type OrgLogoStore interface {
	Put(ctx context.Context, mode OrgLogoMode, content io.ReadSeeker) error
	Get(ctx context.Context, mode OrgLogoMode) (io.ReadCloser, int64, error)
	Delete(ctx context.Context, mode OrgLogoMode) error
	Exists(ctx context.Context, mode OrgLogoMode) (bool, error)
}
