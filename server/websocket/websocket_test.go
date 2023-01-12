package websocket

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/igm/sockjs-go/v3/sockjs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// readOpenMessage reads the sockjs open message
func readOpenMessage(t *testing.T, conn *websocket.Conn) {
	// Read the open message
	mType, data, err := conn.ReadMessage()
	require.Equal(t, websocket.TextMessage, mType)
	require.Nil(t, err)

	require.Equal(t, []byte("o"), data, "expected sockjs open message")
}

// readJSONMessage reads a sockjs JSON message
func readJSONMessage(t *testing.T, conn *websocket.Conn) string {
	mType, data, err := conn.ReadMessage()
	require.Nil(t, err)
	require.Equal(t, websocket.TextMessage, mType)

	assert.Equal(t, "a", string(data[0]), "expected sockjs data frame")

	// Unwrap from sockjs frame
	d := []string{}
	err = json.Unmarshal(data[1:], &d)
	require.Nil(t, err)
	require.Len(t, d, 1)

	return d[0]
}

func writeJSONMessage(t *testing.T, conn *websocket.Conn, typ string, data interface{}) {
	buf, err := json.Marshal(JSONMessage{typ, data})
	require.Nil(t, err)

	// Wrap in sockjs frame
	d, err := json.Marshal([]string{string(buf)})
	require.Nil(t, err)

	// Writes from the client to the server do not include the "a"
	require.NoError(t, conn.WriteMessage(websocket.TextMessage, d))
}

func TestWriteJSONMessage(t *testing.T) {
	cases := []struct {
		typ  string
		data interface{}
	}{
		{
			typ:  "string",
			data: "some string",
		},
		{
			typ:  "map",
			data: map[string]string{"foo": "bar"},
		},
		{
			typ: "struct",
			data: struct {
				Foo int    `json:"foo"`
				Bar string `json:"bar"`
			}{
				Foo: 16,
				Bar: "baz",
			},
		},
	}

	for _, tt := range cases {
		t.Run("", func(t *testing.T) {
			handler := sockjs.NewHandler("/test", sockjs.DefaultOptions, func(session sockjs.Session) {
				defer session.Close(0, "none")

				conn := &Conn{session}

				require.Nil(t, conn.WriteJSONMessage(tt.typ, tt.data))
			})

			srv := httptest.NewServer(handler)
			u, _ := url.Parse(srv.URL)
			u.Scheme = "ws"
			u.Path += "/test/123/abcdefghijklmnop/websocket"

			conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
			require.Nil(t, err)
			defer conn.Close()
			readOpenMessage(t, conn)

			dataJSON, err := json.Marshal(tt.data)
			require.Nil(t, err)

			// Ensure we read the correct message
			data := readJSONMessage(t, conn)
			assert.JSONEq(t,
				fmt.Sprintf(`{"type": "%s", "data": %s}`, tt.typ, dataJSON),
				data,
			)
		})
	}
}

func TestWriteJSONError(t *testing.T) {
	cases := []struct {
		err interface{}
	}{
		{
			err: "this is an error",
		},
		{
			err: struct {
				Error string            `json:"error"`
				Extra map[string]string `json:"extra"`
			}{
				Error: "an error",
				Extra: map[string]string{"foo": "bar"},
			},
		},
	}

	for _, tt := range cases {
		t.Run("", func(t *testing.T) {
			handler := sockjs.NewHandler("/test", sockjs.DefaultOptions, func(session sockjs.Session) {
				defer session.Close(0, "none")

				conn := &Conn{session}

				require.Nil(t, conn.WriteJSONError(tt.err))
			})

			srv := httptest.NewServer(handler)
			u, _ := url.Parse(srv.URL)
			u.Scheme = "ws"
			u.Path += "/test/123/abcdefghijklmnop/websocket"

			conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
			require.Nil(t, err)
			defer conn.Close()
			readOpenMessage(t, conn)

			errJSON, err := json.Marshal(tt.err)
			require.Nil(t, err)

			// Ensure we read the correct message
			data := readJSONMessage(t, conn)
			assert.JSONEq(t,
				fmt.Sprintf(`{"type": "error", "data": %s}`, errJSON),
				data,
			)
		})
	}
}

func TestReadJSONMessage(t *testing.T) {
	cases := []struct {
		typ  string
		data interface{}
		err  error
	}{
		{
			typ:  "string",
			data: "some string",
		},
		{
			typ:  "map",
			data: map[string]string{"foo": "bar"},
		},
		{
			typ: "struct",
			data: struct {
				Foo int    `json:"foo"`
				Bar string `json:"bar"`
			}{
				Foo: 16,
				Bar: "baz",
			},
		},
		{
			typ: "",
			err: errors.New("missing message type"),
		},
	}

	for _, tt := range cases {
		t.Run("", func(t *testing.T) {
			dataJSON, err := json.Marshal(tt.data)
			require.Nil(t, err)

			completed := make(chan struct{})
			handler := sockjs.NewHandler("/test", sockjs.DefaultOptions, func(session sockjs.Session) {
				defer session.Close(0, "none")
				defer func() { completed <- struct{}{} }()

				conn := &Conn{session}

				msg, err := conn.ReadJSONMessage()
				if tt.err == nil {
					require.Nil(t, err)
				} else {
					require.Equal(t, tt.err.Error(), err.Error())
					return
				}

				assert.Equal(t, tt.typ, msg.Type)
				assert.EqualValues(t, &dataJSON, msg.Data)
			})

			// Connect to websocket handler server
			srv := httptest.NewServer(handler)
			u, _ := url.Parse(srv.URL)
			u.Scheme = "ws"
			u.Path += "/test/123/abcdefghijklmnop/websocket"

			conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
			require.Nil(t, err)
			defer conn.Close()

			readOpenMessage(t, conn)

			writeJSONMessage(t, conn, tt.typ, tt.data)

			select {
			case <-completed:
				// Normal
			case <-time.After(1 * time.Second):
				t.Error("handler did not complete")
			}
		})
	}
}

func TestReadAuthToken(t *testing.T) {
	cases := []struct {
		typ   string
		data  authData
		token string
		err   error
	}{
		{
			typ:   "auth",
			data:  authData{Token: "foobar"},
			token: "foobar",
		},
		{
			typ:   "auth",
			data:  authData{Token: ""},
			token: "",
		},
		{
			typ:  "string",
			data: authData{Token: ""},
			err:  errors.New(`message type not "auth": "string"`),
		},
	}

	for _, tt := range cases {
		t.Run("", func(t *testing.T) {
			completed := make(chan struct{})
			handler := sockjs.NewHandler("/test", sockjs.DefaultOptions, func(session sockjs.Session) {
				defer session.Close(0, "none")
				defer func() { completed <- struct{}{} }()

				conn := &Conn{session}

				token, err := conn.ReadAuthToken()
				if tt.err == nil {
					require.Nil(t, err)
				} else {
					require.Equal(t, tt.err.Error(), err.Error())
					return
				}

				assert.EqualValues(t, tt.token, token)
			})

			// Connect to websocket handler server
			srv := httptest.NewServer(handler)
			u, _ := url.Parse(srv.URL)
			u.Scheme = "ws"
			u.Path += "/test/123/abcdefghijklmnop/websocket"

			conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
			require.Nil(t, err)
			defer conn.Close()

			readOpenMessage(t, conn)

			writeJSONMessage(t, conn, tt.typ, tt.data)

			select {
			case <-completed:
				// Normal
			case <-time.After(1 * time.Second):
				t.Error("handler did not complete")
			}
		})
	}
}
