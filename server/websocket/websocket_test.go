package websocket

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimeout(t *testing.T) {
	completed := make(chan struct{})
	handler := func(w http.ResponseWriter, req *http.Request) {
		defer func() { completed <- struct{}{} }()

		conn, err := Upgrade(w, req)
		require.Nil(t, err)
		defer conn.Close()

		conn.Timeout = 1 * time.Millisecond

		_, err = conn.ReadJSONMessage()
		assert.NotNil(t, err, "read should timeout and error")
	}

	// Connect to websocket handler server
	srv := httptest.NewServer(http.HandlerFunc(handler))
	u, _ := url.Parse(srv.URL)
	u.Scheme = "ws"
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.Nil(t, err)
	defer conn.Close()

	select {
	case <-completed:
		// Normal
	case <-time.After(1 * time.Second):
		t.Error("handler did not complete")
	}
}

func TestWriteJSONMessage(t *testing.T) {
	var cases = []struct {
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
			handler := func(w http.ResponseWriter, req *http.Request) {
				conn, err := Upgrade(w, req)
				require.Nil(t, err)
				defer conn.Close()

				require.Nil(t, conn.WriteJSONMessage(tt.typ, tt.data))
			}

			// Connect to websocket handler server
			srv := httptest.NewServer(http.HandlerFunc(handler))
			u, _ := url.Parse(srv.URL)
			u.Scheme = "ws"
			conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
			require.Nil(t, err)
			defer conn.Close()

			dataJSON, err := json.Marshal(tt.data)
			require.Nil(t, err)

			// Ensure we read the correct message
			mType, data, err := conn.ReadMessage()
			require.Nil(t, err)
			assert.Equal(t, websocket.TextMessage, mType)
			assert.JSONEq(t,
				fmt.Sprintf(`{"type": "%s", "data": %s}`, tt.typ, dataJSON),
				string(data),
			)

		})
	}
}

func TestWriteJSONError(t *testing.T) {
	var cases = []struct {
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
			handler := func(w http.ResponseWriter, req *http.Request) {
				conn, err := Upgrade(w, req)
				require.Nil(t, err)
				defer conn.Close()

				require.Nil(t, conn.WriteJSONError(tt.err))
			}

			// Connect to websocket handler server
			srv := httptest.NewServer(http.HandlerFunc(handler))
			u, _ := url.Parse(srv.URL)
			u.Scheme = "ws"
			conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
			require.Nil(t, err)
			defer conn.Close()

			errJSON, err := json.Marshal(tt.err)
			require.Nil(t, err)

			// Ensure we read the correct message
			mType, data, err := conn.ReadMessage()
			require.Nil(t, err)
			assert.Equal(t, websocket.TextMessage, mType)
			assert.JSONEq(t,
				fmt.Sprintf(`{"type": "error", "data": %s}`, errJSON),
				string(data),
			)

		})
	}
}

func TestReadJSONMessage(t *testing.T) {
	var cases = []struct {
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
			handler := func(w http.ResponseWriter, req *http.Request) {
				defer func() { completed <- struct{}{} }()

				conn, err := Upgrade(w, req)
				require.Nil(t, err)
				defer conn.Close()

				msg, err := conn.ReadJSONMessage()
				if tt.err == nil {
					require.Nil(t, err)
				} else {
					require.Equal(t, tt.err.Error(), err.Error())
					return
				}

				assert.Equal(t, tt.typ, msg.Type)
				assert.EqualValues(t, &dataJSON, msg.Data)

			}

			// Connect to websocket handler server
			srv := httptest.NewServer(http.HandlerFunc(handler))
			u, _ := url.Parse(srv.URL)
			u.Scheme = "ws"
			wsConn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
			require.Nil(t, err)
			conn := &Conn{wsConn, defaultTimeout}
			defer conn.Close()

			require.Nil(t, conn.WriteJSONMessage(tt.typ, tt.data))

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
	var cases = []struct {
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
			handler := func(w http.ResponseWriter, req *http.Request) {
				defer func() { completed <- struct{}{} }()

				conn, err := Upgrade(w, req)
				require.Nil(t, err)
				defer conn.Close()

				token, err := conn.ReadAuthToken()
				if tt.err == nil {
					require.Nil(t, err)
				} else {
					require.Equal(t, tt.err.Error(), err.Error())
					return
				}

				assert.Equal(t, tt.token, token)
			}

			// Connect to websocket handler server
			srv := httptest.NewServer(http.HandlerFunc(handler))
			u, _ := url.Parse(srv.URL)
			u.Scheme = "ws"
			wsConn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
			require.Nil(t, err)
			conn := &Conn{wsConn, defaultTimeout}
			defer conn.Close()

			require.Nil(t, conn.WriteJSONMessage(tt.typ, tt.data))

			select {
			case <-completed:
				// Normal
			case <-time.After(1 * time.Second):
				t.Error("handler did not complete")
			}
		})
	}
}
