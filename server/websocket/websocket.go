// Package websocket contains helpers and implementation for backend functions
// that interact with the frontend over websockets.
package websocket

import (
	"encoding/json"

	"github.com/igm/sockjs-go/v3/sockjs"

	"github.com/fleetdm/fleet/v4/server/contexts/token"
	"github.com/pkg/errors"
)

const (
	// authType is the type string used for auth messages.
	authType string = "auth"

	// errType is the type string used for error messages.
	errType string = "error"
)

// JSONMessage is a wrapper struct for messages that will be sent across the wire
// as JSON.
type JSONMessage struct {
	// Type is a string indicating which message type the data contains
	Type string `json:"type"`
	// Data contains the arbitrarily schemaed JSON data. Type should
	// indicate how this should be deserialized.
	Data interface{} `json:"data"`
}

// Conn is a wrapper for a standard websocket connection with utility methods
// added for interacting with Fleet specific message types.
type Conn struct {
	sockjs.Session
}

func (c *Conn) WriteJSON(msg JSONMessage) error {
	buf, err := json.Marshal(msg)
	if err != nil {
		return errors.Wrap(err, "marshalling JSON")
	}
	return errors.Wrap(c.Send(string(buf)), "sending")
}

// WriteJSONMessage writes the provided data as JSON (using the Message struct),
// returning any error condition from the connection.
func (c *Conn) WriteJSONMessage(typ string, data interface{}) error {
	return c.WriteJSON(JSONMessage{Type: typ, Data: data})
}

// WriteJSONError writes an error (Message struct with Type="error"), returning any
// error condition from the connection.
func (c *Conn) WriteJSONError(data interface{}) error {
	return c.WriteJSONMessage(errType, data)
}

// ReadJSONMessage reads an incoming Message from JSON. Note that the
// Message.Data field is guaranteed to be *json.RawMessage, and so unchecked
// type assertions may be performed as in:
//  msg, err := c.ReadJSONMessage()
//  if err == nil && msg.Type == "foo" {
//  	var foo fooData
//  	json.Unmarshal(*(msg.Data.(*json.RawMessage)), &foo)
//  }
func (c *Conn) ReadJSONMessage() (*JSONMessage, error) {
	data, err := c.Recv()
	if err != nil {
		return nil, errors.Wrap(err, "reading from websocket")
	}

	msg := &JSONMessage{Data: &json.RawMessage{}}

	if err := json.Unmarshal([]byte(data), msg); err != nil {
		return nil, errors.Wrap(err, "parsing msg json")
	}

	if msg.Type == "" {
		return nil, errors.New("missing message type")
	}

	return msg, nil
}

// authData defines the data used to authenticate a Fleet frontend client over
// a websocket connection.
type authData struct {
	Token token.Token `json:"token"`
}

// ReadAuthToken reads from the websocket, returning an auth token embedded in
// a JSONMessage with type "auth" and data that can be unmarshalled to
// authData.
func (c *Conn) ReadAuthToken() (token.Token, error) {
	msg, err := c.ReadJSONMessage()
	if err != nil {
		return "", errors.Wrap(err, "read auth token")
	}
	if msg.Type != authType {
		return "", errors.Errorf(`message type not "%s": "%s"`, authType, msg.Type)
	}

	var auth authData
	if err := json.Unmarshal(*(msg.Data.(*json.RawMessage)), &auth); err != nil {
		return "", errors.Wrap(err, "unmarshal auth data")
	}

	return auth.Token, nil
}
