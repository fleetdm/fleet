package mdm

import "encoding/hex"

type hexData []byte

// String returns the hex-encoded string form of h
func (h hexData) String() string {
	return hex.EncodeToString(h)
}

// Push contains data needed to send an APNs push to MDM enrollments.
type Push struct {
	PushMagic string
	Token     hexData
	Topic     string
}

// SetTokenString decodes the hex-encoded token into p
func (p *Push) SetTokenString(token string) (err error) {
	p.Token, err = hex.DecodeString(token)
	return
}
