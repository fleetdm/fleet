package cached_mysql

import (
	"encoding/json"
	"maps"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

type packsList []*fleet.Pack

func (pl packsList) Clone() (fleet.Cloner, error) {
	var cloned packsList
	if pl == nil {
		return cloned, nil
	}

	cloned = make(packsList, 0, len(pl))
	for _, p := range pl {
		cloned = append(cloned, p.Copy())
	}
	return cloned, nil
}

// microsoftGraphCredentialsList is the cacheable form of the Graph credential list. Cloning deep-copies each entry,
// including the LastSyncedAt/LastSyncError pointers, so a caller mutating what it read cannot corrupt the cache.
type microsoftGraphCredentialsList []*fleet.MicrosoftGraphCredential

func (cl microsoftGraphCredentialsList) Clone() (fleet.Cloner, error) {
	var cloned microsoftGraphCredentialsList
	if cl == nil {
		return cloned, nil
	}

	cloned = make(microsoftGraphCredentialsList, 0, len(cl))
	for _, c := range cl {
		clone, err := c.Clone()
		if err != nil {
			return nil, err
		}
		cloned = append(cloned, clone.(*fleet.MicrosoftGraphCredential))
	}
	return cloned, nil
}

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

type integer int

func (i integer) Clone() (fleet.Cloner, error) {
	return i, nil
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
