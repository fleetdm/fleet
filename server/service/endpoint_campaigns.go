package service

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/fleetdm/fleet/v4/server/config"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/websocket"
	kitlog "github.com/go-kit/log"
	gws "github.com/gorilla/websocket"
	"github.com/igm/sockjs-go/v3/sockjs"
)

////////////////////////////////////////////////////////////////////////////////
// Stream Distributed Query Campaign Results and Metadata
////////////////////////////////////////////////////////////////////////////////

var reVersion = regexp.MustCompile(`\{fleetversion:\(\?:([^\}\)]+)\)\}`)

func makeStreamDistributedQueryCampaignResultsHandler(config config.ServerConfig, svc fleet.Service, logger kitlog.Logger) func(string) http.Handler {
	opt := sockjs.DefaultOptions
	opt.Websocket = true
	opt.RawWebsocket = true

	if config.WebsocketsAllowUnsafeOrigin {
		opt.CheckOrigin = func(r *http.Request) bool {
			return true
		}
		// sockjs uses gorilla websockets under-the-hood see https://github.com/igm/sockjs-go/blob/master/v3/sockjs/rawwebsocket.go#L12-L14
		opt.WebsocketUpgrader = &gws.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			}}
	}

	return func(path string) http.Handler {
		// expand the path's versions (with regex) to all literal paths (no regex),
		// because sockjs requires the (static, literal) path prefix as argument to
		// create the handler so that it can trim it from the request's URL to get
		// the special path values (such as the session id).
		matches := reVersion.FindStringSubmatch(path)
		if len(matches) == 0 {
			panic("unexpected path, could not expand fleetversion: " + path)
		}

		versions := strings.Split(matches[1], "|")
		literalPaths := make([]string, len(versions))
		for i, ver := range versions {
			lp := reVersion.ReplaceAllStringFunc(path, func(_ string) string { return ver })
			literalPaths[i] = lp
		}

		sockHandler := func(session sockjs.Session) {
			conn := &websocket.Conn{Session: session}
			defer func() {
				if p := recover(); p != nil {
					logger.Log("err", p, "msg", "panic in result handler")
					conn.WriteJSONError("panic in result handler") //nolint:errcheck
				}
				session.Close(0, "none")
			}()

			// Receive the auth bearer token
			token, err := conn.ReadAuthToken()
			if err != nil {
				logger.Log("err", err, "msg", "failed to read auth token")
				return
			}

			// Authenticate with the token
			vc, err := authViewer(context.Background(), string(token), svc)
			if err != nil || !vc.CanPerformActions() {
				logger.Log("err", err, "msg", "unauthorized viewer")
				conn.WriteJSONError("unauthorized") //nolint:errcheck
				return
			}

			ctx := viewer.NewContext(context.Background(), *vc)

			msg, err := conn.ReadJSONMessage()
			if err != nil {
				logger.Log("err", err, "msg", "reading select_campaign JSON")
				conn.WriteJSONError("error reading select_campaign") //nolint:errcheck
				return
			}
			if msg.Type != "select_campaign" {
				logger.Log("err", "unexpected msg type, expected select_campaign", "msg-type", msg.Type)
				conn.WriteJSONError("expected select_campaign") //nolint:errcheck
				return
			}

			var info struct {
				CampaignID uint `json:"campaign_id"`
			}
			err = json.Unmarshal(*(msg.Data.(*json.RawMessage)), &info)
			if err != nil {
				logger.Log("err", err, "msg", "unmarshaling select_campaign data")
				conn.WriteJSONError("error unmarshaling select_campaign data") //nolint:errcheck
				return
			}
			if info.CampaignID == 0 {
				logger.Log("err", "campaign ID not set")
				conn.WriteJSONError("0 is not a valid campaign ID") //nolint:errcheck
				return
			}

			svc.StreamCampaignResults(ctx, conn, info.CampaignID)
		}

		// multiplex the requests to each literal path that this endpoint support,
		// with the corresponding sockjs handler to handle that specific path.
		mux := http.NewServeMux()
		for _, lp := range literalPaths {
			// important: sockjs' path must not have the trailing path, but the mux
			// needs it in order to match it as a path prefix (subtree).
			sockPath := strings.TrimSuffix(lp, "/")
			mux.Handle(lp, sockjs.NewHandler(sockPath, opt, sockHandler))
		}
		return mux
	}
}
