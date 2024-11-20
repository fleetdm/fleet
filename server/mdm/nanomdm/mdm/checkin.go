package mdm

import (
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/groob/plist"
)

var ErrUnrecognizedMessageType = errors.New("unrecognized MessageType")

// MessageType represents the MessageType of a check-in message
type MessageType struct {
	MessageType string
}

// Authenticate is a representation of an "Authenticate" check-in message type.
// See https://developer.apple.com/documentation/devicemanagement/authenticaterequest
type Authenticate struct {
	Enrollment
	MessageType
	Topic string
	Raw   []byte `plist:"-"` // Original Authenticate XML plist

	// Additional fields required in AuthenticateRequest as specified
	// in the Apple documentation.
	DeviceName string
	Model      string
	ModelName  string

	// ProductName contains the device's product name (e.g. `iPhone3,1`).
	//
	// iPhones and iPads send ProductName but not Model/ModelName,
	// thus we use this field as the device's Model (which is required
	// on lifecycle stages).
	ProductName string

	// Fields that may be present but are not strictly required for the
	// operation of the MDM protocol. Nice-to-haves.
	SerialNumber string `plist:",omitempty"`
}

type b64Data []byte

// String returns the base64-encoded string form of b
func (b b64Data) String() string {
	return base64.StdEncoding.EncodeToString(b)
}

// TokenUpdate is a representation of a "TokenUpdate" check-in message type.
// See https://developer.apple.com/documentation/devicemanagement/token_update
type TokenUpdate struct {
	Enrollment
	MessageType
	Push
	UnlockToken []byte `plist:",omitempty"`
	Raw         []byte `plist:"-"` // Original TokenUpdate XML plist
}

// CheckOut is a representation of a "CheckOut" check-in message type.
// See https://developer.apple.com/documentation/devicemanagement/checkoutrequest
type CheckOut struct {
	Enrollment
	MessageType
	Raw []byte `plist:"-"` // Original CheckOut XML plist
}

// UserAuthenticate is a representation of a "UserAuthenticate" check-in message type.
// https://developer.apple.com/documentation/devicemanagement/userauthenticaterequest
type UserAuthenticate struct {
	Enrollment
	MessageType
	DigestResponse string `plist:",omitempty"`
	Raw            []byte `plist:"-"` // Original XML plist
}

type BootstrapToken struct {
	BootstrapToken b64Data
}

// SetTokenString decodes the base64-encoded bootstrap token into t
func (t *BootstrapToken) SetTokenString(token string) error {
	tokenRaw, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return err
	}
	t.BootstrapToken = tokenRaw
	return nil
}

// SetBootstrapToken is a representation of a "SetBootstrapToken" check-in message type.
// See https://developer.apple.com/documentation/devicemanagement/setbootstraptokenrequest
type SetBootstrapToken struct {
	Enrollment
	MessageType
	BootstrapToken
	Raw []byte `plist:"-"` // Original XML plist
}

// GetBootstrapToken is a representation of a "GetBootstrapToken" check-in message type.
// See https://developer.apple.com/documentation/devicemanagement/getbootstraptokenrequest
type GetBootstrapToken struct {
	Enrollment
	MessageType
	Raw []byte `plist:"-"` // Original XML plist
}

// DeclarativeManagement is a representation of a "DeclarativeManagement" check-in message type.
// See https://developer.apple.com/documentation/devicemanagement/declarativemanagementrequest
type DeclarativeManagement struct {
	Enrollment
	MessageType
	Data     []byte
	Endpoint string
	Raw      []byte `plist:"-"` // Original XML plist
}

// TokenParameters is a representation of a "GetTokenRequest.TokenParameters" structure.
// See https://developer.apple.com/documentation/devicemanagement/gettokenrequest/tokenparameters
type TokenParameters struct {
	PhoneUDID     string
	SecurityToken string
	WatchUDID     string
}

// GetTokenResponse is a representation of a "GetTokenResponse" structure.
// See https://developer.apple.com/documentation/devicemanagement/gettokenresponse
type GetTokenResponse struct {
	TokenData []byte
}

// GetToken is a representation of a "GetToken" check-in message type.
// See https://developer.apple.com/documentation/devicemanagement/get_token
type GetToken struct {
	Enrollment
	MessageType
	TokenServiceType string
	TokenParameters  *TokenParameters `plist:",omitempty"`
	Raw              []byte           `plist:"-"` // Original XML plist
}

// Validate validates a GetToken check-in message.
func (m *GetToken) Validate() error {
	if m == nil {
		return errors.New("nil GetToken")
	}
	if m.TokenServiceType == "" {
		return errors.New("empty GetToken TokenServiceType")
	}
	if m.TokenServiceType == "com.apple.watch.pairing" && m.TokenParameters == nil {
		return fmt.Errorf("nil TokenParameters for GetToken: %s", m.TokenServiceType)
	}
	return nil
}

// newCheckinMessageForType returns a pointer to a check-in struct for MessageType t
func newCheckinMessageForType(t string, raw []byte) interface{} {
	switch t {
	case "Authenticate":
		return &Authenticate{Raw: raw}
	case "TokenUpdate":
		return &TokenUpdate{Raw: raw}
	case "CheckOut":
		return &CheckOut{Raw: raw}
	case "SetBootstrapToken":
		return &SetBootstrapToken{Raw: raw}
	case "GetBootstrapToken":
		return &GetBootstrapToken{Raw: raw}
	case "UserAuthenticate":
		return &UserAuthenticate{Raw: raw}
	case "DeclarativeManagement":
		return &DeclarativeManagement{Raw: raw}
	case "GetToken":
		return &GetToken{Raw: raw}
	default:
		return nil
	}
}

// checkinUnmarshaller facilitates unmarshalling a plist check-in message.
type checkinUnmarshaller struct {
	message interface{}
	raw     []byte
}

// UnmarshalPlist populates the message field of w based on the contents of a plist.
func (w *checkinUnmarshaller) UnmarshalPlist(f func(interface{}) error) error {
	onlyType := new(MessageType)
	err := f(onlyType)
	if err != nil {
		return err
	}
	w.message = newCheckinMessageForType(onlyType.MessageType, w.raw)
	if w.message == nil {
		return fmt.Errorf("%w: %q", ErrUnrecognizedMessageType, onlyType.MessageType)
	}
	return f(w.message)
}

// DecodeCheckin unmarshals rawMessage into a specific check-in struct in message.
func DecodeCheckin(rawMessage []byte) (message interface{}, err error) {
	w := &checkinUnmarshaller{raw: rawMessage}
	err = plist.Unmarshal(rawMessage, w)
	message = w.message
	return
}
