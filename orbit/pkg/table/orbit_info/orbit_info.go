package orbit_info

import (
	"context"
	"strconv"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/build"
	orbit_table "github.com/fleetdm/fleet/v4/orbit/pkg/table"
	"github.com/fleetdm/fleet/v4/orbit/pkg/token"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/osquery/osquery-go/plugin/table"
)

// Extension implements an extension table that provides info about Orbit.
type Extension struct {
	startTime       time.Time
	orbitClient     *service.OrbitClient
	orbitChannel    string
	osquerydChannel string
	desktopChannel  string
	dektopVersion   string
	trw             *token.ReadWriter
	scriptsEnabled  func() bool
	updateURL       string
}

var _ orbit_table.Extension = (*Extension)(nil)

func New(
	orbitClient *service.OrbitClient, orbitChannel, osquerydChannel, desktopChannel string, desktopVersion string, trw *token.ReadWriter,
	startTime time.Time, scriptsEnabled func() bool,
	updateURL string,
) *Extension {
	return &Extension{
		startTime:       startTime,
		orbitClient:     orbitClient,
		orbitChannel:    orbitChannel,
		osquerydChannel: osquerydChannel,
		desktopChannel:  desktopChannel,
		dektopVersion:   desktopVersion,
		trw:             trw,
		scriptsEnabled:  scriptsEnabled,
		updateURL:       updateURL,
	}
}

// Name partially implements orbit_table.Extension.
func (o Extension) Name() string {
	return "orbit_info"
}

// Columns partially implements orbit_table.Extension.
func (o Extension) Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("version"),
		table.TextColumn("device_auth_token"),
		table.TextColumn("enrolled"),
		table.TextColumn("last_recorded_error"),
		table.TextColumn("orbit_channel"),
		table.TextColumn("osqueryd_channel"),
		table.TextColumn("desktop_channel"),
		table.TextColumn("desktop_version"),
		table.BigIntColumn("uptime"),
		table.IntegerColumn("scripts_enabled"),
		table.TextColumn("update_url"),
	}
}

// GenerateFunc partially implements table.Extension.
func (o Extension) GenerateFunc(_ context.Context, _ table.QueryContext) ([]map[string]string, error) {
	v := build.Version
	if v == "" {
		v = "unknown"
	}
	lastRecordedError := ""
	if err := o.orbitClient.LastRecordedError(); err != nil {
		lastRecordedError = err.Error()
	}

	var err error
	var token string
	if o.trw != nil {
		if token, err = o.trw.Read(); err != nil {
			return nil, err
		}
	}

	boolToInt := func(b bool) int64 {
		// Fast implementation according to https://0x0f.me/blog/golang-compiler-optimization/
		var i int64
		if b {
			i = 1
		} else {
			i = 0
		}
		return i
	}

	return []map[string]string{{
		"version":             v,
		"device_auth_token":   token,
		"enrolled":            strconv.FormatBool(o.orbitClient.Enrolled()),
		"last_recorded_error": lastRecordedError,
		"orbit_channel":       o.orbitChannel,
		"osqueryd_channel":    o.osquerydChannel,
		"desktop_channel":     o.desktopChannel,
		"desktop_version":     o.dektopVersion,
		"uptime":              strconv.FormatInt(int64(time.Since(o.startTime).Seconds()), 10),
		"scripts_enabled":     strconv.FormatInt(boolToInt(o.scriptsEnabled()), 10),
		"update_url":          o.updateURL,
	}}, nil
}
