package cached_mysql

import (
	"encoding/json"
	"maps"
	"slices"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

type rawJSONMessage json.RawMessage

func (r *rawJSONMessage) Clone() (fleet.Cloner, error) {
	var clone *rawJSONMessage
	if r == nil {
		return clone, nil
	}

	msg := make(rawJSONMessage, len(*r))
	copy(msg, *r)
	return &msg, nil
}

type queryIDList []uint

func (ql queryIDList) Clone() (fleet.Cloner, error) {
	return slices.Clone(ql), nil
}

// fmaNameMap is a map of unique_identifier -> canonical FMA name.
// Used during software ingestion to override osquery-reported names.
type fmaNameMap map[string]string

func (m fmaNameMap) Clone() (fleet.Cloner, error) {
	if m == nil {
		return fmaNameMap(nil), nil
	}
	clone := make(fmaNameMap, len(m))
	maps.Copy(clone, m)
	return clone, nil
}
