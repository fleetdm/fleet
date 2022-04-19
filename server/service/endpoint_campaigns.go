package service

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/websocket"
	kitlog "github.com/go-kit/kit/log"
	"github.com/igm/sockjs-go/v3/sockjs"
)

////////////////////////////////////////////////////////////////////////////////
// Stream Distributed Query Campaign Results and Metadata
////////////////////////////////////////////////////////////////////////////////

var reVersion = regexp.MustCompile(`\{fleetversion:\(\?:([^\}\)]+)\)\}`)

func makeStreamDistributedQueryCampaignResultsHandler(svc fleet.Service, logger kitlog.Logger) func(string) http.Handler {
	opt := sockjs.DefaultOptions
	opt.Websocket = true
	opt.RawWebsocket = true

	return func(path string) http.Handler {
		// expand the path's versions (with regex) to all literal paths (no regex)
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
					conn.WriteJSONError("panic in result handler")
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
				conn.WriteJSONError("unauthorized")
				return
			}

			ctx := viewer.NewContext(context.Background(), *vc)

			msg, err := conn.ReadJSONMessage()
			if err != nil {
				logger.Log("err", err, "msg", "reading select_campaign JSON")
				conn.WriteJSONError("error reading select_campaign")
				return
			}
			if msg.Type != "select_campaign" {
				logger.Log("err", "unexpected msg type, expected select_campaign", "msg-type", msg.Type)
				conn.WriteJSONError("expected select_campaign")
				return
			}

			var info struct {
				CampaignID uint `json:"campaign_id"`
			}
			err = json.Unmarshal(*(msg.Data.(*json.RawMessage)), &info)
			if err != nil {
				logger.Log("err", err, "msg", "unmarshaling select_campaign data")
				conn.WriteJSONError("error unmarshaling select_campaign data")
				return
			}
			if info.CampaignID == 0 {
				logger.Log("err", "campaign ID not set")
				conn.WriteJSONError("0 is not a valid campaign ID")
				return
			}

			svc.StreamCampaignResults(ctx, conn, info.CampaignID)
		}

		mux := http.NewServeMux()
		for _, lp := range literalPaths {
			sockPath := strings.TrimSuffix(lp, "/")
			mux.Handle(lp, sockjs.NewHandler(sockPath, opt, sockHandler))
		}
		return mux
	}
}
