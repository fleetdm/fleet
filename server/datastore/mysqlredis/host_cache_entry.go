package mysqlredis

import (
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// hostCacheEnvelope is the JSON wire format for cached host lookups. It
// embeds fleet.Host so every normally-serializable field rides along
// automatically, then shadows the four fields fleet.Host tags `json:"-"` to
// keep out of HTTP responses: OsqueryHostID, NodeKey, OrbitNodeKey, and
// HasHostIdentityCert. These four MUST round-trip or auth breaks (a cached
// host with HasHostIdentityCert=nil would cause AuthenticateHost to skip the
// httpsig check for up to TTL).
//
// Why embedding works without collision: the embedded fleet.Host has those
// four fields tagged `json:"-"`, so encoding/json skips them entirely. Our
// outer fields carry the real JSON names. On unmarshal, the tagged JSON keys
// map to the outer fields; toHost() then copies them back onto the embedded
// Host so downstream code can read them in their natural positions.
//
// One envelope serves both LoadHostByNodeKey and LoadHostByOrbitNodeKey
// because their SELECT lists differ only in which fleet.Host fields they
// populate; unpopulated pointer/slice fields fall out via omitempty, and
// the handful of non-pointer orbit-specific fields (MDM.EncryptionKeyAvailable)
// are small enough that the constant overhead doesn't matter.
//
// When fleet.Host gains a new `json:"-"` field that downstream auth code
// reads, add a shadow here in lockstep. TestPBT_HostCacheEnvelopeRoundTrip
// catches drift by asserting full-struct equivalence after marshal/unmarshal.
type hostCacheEnvelope struct {
	fleet.Host

	OsqueryHostID       *string `json:"osquery_host_id,omitempty"`
	NodeKey             *string `json:"node_key,omitempty"`
	OrbitNodeKey        *string `json:"orbit_node_key,omitempty"`
	HasHostIdentityCert *bool   `json:"has_host_identity_cert,omitempty"`
}

// envelopeFromHost builds an envelope suitable for JSON marshaling by copying
// the four json:"-" shadow fields out of the embedded Host. Caller must
// ensure h is non-nil.
func envelopeFromHost(h *fleet.Host) *hostCacheEnvelope {
	return &hostCacheEnvelope{
		Host:                *h,
		OsqueryHostID:       h.OsqueryHostID,
		NodeKey:             h.NodeKey,
		OrbitNodeKey:        h.OrbitNodeKey,
		HasHostIdentityCert: h.HasHostIdentityCert,
	}
}

// toHost returns a fresh *fleet.Host populated from the envelope, with the
// shadow fields copied back onto the embedded Host so downstream auth code
// reads them in their natural positions.
func (e *hostCacheEnvelope) toHost() *fleet.Host {
	h := e.Host
	h.OsqueryHostID = e.OsqueryHostID
	h.NodeKey = e.NodeKey
	h.OrbitNodeKey = e.OrbitNodeKey
	h.HasHostIdentityCert = e.HasHostIdentityCert
	return &h
}
