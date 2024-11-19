package api

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/cryptoutil"
	mdmhttp "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/http"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/storage"

	"github.com/micromdm/nanolib/log"
	"github.com/micromdm/nanolib/log/ctxlog"
)

// enrolledAPIResult is a per-enrollment API result.
type enrolledAPIResult struct {
	PushError    string `json:"push_error,omitempty"`
	PushResult   string `json:"push_result,omitempty"`
	CommandError string `json:"command_error,omitempty"`
}

// enrolledAPIResults is a map of enrollments to a per-enrollment API result.
type enrolledAPIResults map[string]*enrolledAPIResult

// apiResult is the JSON reply returned from either pushing or queuing commands.
type apiResult struct {
	Status       enrolledAPIResults `json:"status,omitempty"`
	NoPush       bool               `json:"no_push,omitempty"`
	PushError    string             `json:"push_error,omitempty"`
	CommandError string             `json:"command_error,omitempty"`
	CommandUUID  string             `json:"command_uuid,omitempty"`
	RequestType  string             `json:"request_type,omitempty"`
}

type (
	ctxKeyIDFirst struct{}
	ctxKeyIDCount struct{}
)

func setAPIIDs(ctx context.Context, idFirst string, idCount int) context.Context {
	ctx = context.WithValue(ctx, ctxKeyIDFirst{}, idFirst)
	return context.WithValue(ctx, ctxKeyIDCount{}, idCount)
}

func ctxKVs(ctx context.Context) (out []interface{}) {
	id, ok := ctx.Value(ctxKeyIDFirst{}).(string)
	if ok {
		out = append(out, "id_first", id)
	}
	eType, ok := ctx.Value(ctxKeyIDCount{}).(int)
	if ok {
		out = append(out, "id_count", eType)
	}
	return
}

func setupCtxLog(ctx context.Context, ids []string, logger log.Logger) (context.Context, log.Logger) {
	if len(ids) > 0 {
		ctx = setAPIIDs(ctx, ids[0], len(ids))
		ctx = ctxlog.AddFunc(ctx, ctxKVs)
	}
	return ctx, ctxlog.Logger(ctx, logger)
}

// PushHandler sends APNs push notifications to MDM enrollments.
//
// Note the whole URL path is used as the identifier to push to. This
// probably necessitates stripping the URL prefix before using. Also
// note we expose Go errors to the output as this is meant for "API"
// users.
func PushHandler(pusher push.Pusher, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ids := strings.Split(r.URL.Path, ",")
		ctx, logger := setupCtxLog(r.Context(), ids, logger)
		output := apiResult{
			Status: make(enrolledAPIResults),
		}
		logs := []interface{}{"msg", "push"}
		pushResp, err := pusher.Push(ctx, ids)
		if err != nil {
			logs = append(logs, "err", err)
			output.PushError = err.Error()
		}
		var ct, errCt int
		for id, resp := range pushResp {
			output.Status[id] = &enrolledAPIResult{
				PushResult: resp.Id,
			}
			if resp.Err != nil {
				output.Status[id].PushError = resp.Err.Error()
				errCt += 1
			} else {
				ct += 1
			}
		}
		logs = append(logs, "count", ct)
		if errCt > 0 {
			logs = append(logs, "errs", errCt)
		}
		if err != nil || errCt > 0 {
			logger.Info(logs...)
		} else {
			logger.Debug(logs...)
		}
		// generate response codes depending on if everything succeeded, failed, or parially succedded
		header := http.StatusInternalServerError
		if (errCt > 0 || err != nil) && ct > 0 {
			header = http.StatusMultiStatus
		} else if (errCt == 0 && err == nil) && ct >= 1 {
			header = http.StatusOK
		}
		json, err := json.MarshalIndent(output, "", "\t")
		if err != nil {
			logger.Info("msg", "marshal json", "err", err)
		}
		w.Header().Set("Content-type", "application/json")
		w.WriteHeader(header)
		_, err = w.Write(json)
		if err != nil {
			logger.Info("msg", "writing body", "err", err)
		}
	}
}

// RawCommandEnqueueHandler enqueues a raw MDM command plist and sends
// push notifications to MDM enrollments.
//
// Note the whole URL path is used as the identifier to enqueue (and
// push to. This probably necessitates stripping the URL prefix before
// using. Also note we expose Go errors to the output as this is meant
// for "API" users.
func RawCommandEnqueueHandler(enqueuer storage.CommandEnqueuer, pusher push.Pusher, logger log.Logger) http.HandlerFunc {
	if enqueuer == nil {
		panic("nil enqueuer")
	}
	if logger == nil {
		panic("nil logger")
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ids := strings.Split(r.URL.Path, ",")
		ctx, logger := setupCtxLog(r.Context(), ids, logger)
		b, err := mdmhttp.ReadAllAndReplaceBody(r)
		if err != nil {
			logger.Info("msg", "reading body", "err", err)
			var toErr interface{ Timeout() bool }
			if errors.As(err, &toErr) && toErr.Timeout() {
				http.Error(w, http.StatusText(http.StatusRequestTimeout), http.StatusRequestTimeout)
				return
			}
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		command, err := mdm.DecodeCommand(b)
		if err != nil {
			logger.Info("msg", "decoding command", "err", err)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		nopush := r.URL.Query().Get("nopush") != ""
		output := apiResult{
			Status:      make(enrolledAPIResults),
			NoPush:      nopush,
			CommandUUID: command.CommandUUID,
			RequestType: command.Command.RequestType,
		}
		logger = logger.With(
			"command_uuid", command.CommandUUID,
			"request_type", command.Command.RequestType,
		)
		logs := []interface{}{
			"msg", "enqueue",
		}
		idErrs, err := enqueuer.EnqueueCommand(ctx, ids, command)
		ct := len(ids) - len(idErrs)
		if err != nil {
			logs = append(logs, "err", err)
			output.CommandError = err.Error()
			if len(idErrs) == 0 {
				// we assume if there were no ID-specific errors but
				// there was a general error then all IDs failed
				ct = 0
			}
		}
		logs = append(logs, "count", ct)
		if len(idErrs) > 0 {
			logs = append(logs, "errs", len(idErrs))
		}
		if err != nil || len(idErrs) > 0 {
			logger.Info(logs...)
		} else {
			logger.Debug(logs...)
		}
		// loop through our command errors, if any, and add to output
		for id, err := range idErrs {
			if err != nil {
				output.Status[id] = &enrolledAPIResult{
					CommandError: err.Error(),
				}
			}
		}
		// optionally send pushes
		pushResp := make(map[string]*push.Response)
		var pushErr error
		if !nopush && pusher != nil {
			pushResp, pushErr = pusher.Push(ctx, ids)
			if err != nil {
				logger.Info("msg", "push", "err", err)
				output.PushError = err.Error()
			}
		} else if !nopush && pusher == nil {
			pushErr = errors.New("nil pusher")
		}
		// loop through our push errors, if any, and add to output
		var pushCt, pushErrCt int
		for id, resp := range pushResp {
			if _, ok := output.Status[id]; ok {
				output.Status[id].PushResult = resp.Id
			} else {
				output.Status[id] = &enrolledAPIResult{
					PushResult: resp.Id,
				}
			}
			if resp.Err != nil {
				output.Status[id].PushError = resp.Err.Error()
				pushErrCt++
			} else {
				pushCt++
			}
		}
		logs = []interface{}{
			"msg", "push",
			"count", pushCt,
		}
		if pushErr != nil {
			logs = append(logs, "err", pushErr)
		}
		if pushErrCt > 0 {
			logs = append(logs, "errs", pushErrCt)
		}
		if pushErr != nil || pushErrCt > 0 {
			logger.Info(logs...)
		} else {
			logger.Debug(logs...)
		}
		// generate response codes depending on if everything succeeded, failed, or parially succedded
		header := http.StatusInternalServerError
		if (len(idErrs) > 0 || err != nil || (!nopush && (pushErrCt > 0 || pushErr != nil))) && (ct > 0 || (!nopush && (pushCt > 0))) {
			header = http.StatusMultiStatus
		} else if (len(idErrs) == 0 && err == nil && (nopush || (pushErrCt == 0 && pushErr == nil))) && (ct >= 1 && (nopush || (pushCt >= 1))) {
			header = http.StatusOK
		}
		json, err := json.MarshalIndent(output, "", "\t")
		if err != nil {
			logger.Info("msg", "marshal json", "err", err)
		}
		w.Header().Set("Content-type", "application/json")
		w.WriteHeader(header)
		_, err = w.Write(json)
		if err != nil {
			logger.Info("msg", "writing body", "err", err)
		}
	}
}

// readPEMCertAndKey reads a PEM-encoded certificate and non-encrypted
// private key from input bytes and returns the separate PEM certificate
// and private key in cert and key respectively.
func readPEMCertAndKey(input []byte) (cert []byte, key []byte, err error) {
	// if the PEM blocks are mushed together with no newline then add one
	input = bytes.ReplaceAll(input, []byte("----------"), []byte("-----\n-----"))
	var block *pem.Block
	for {
		block, input = pem.Decode(input)
		if block == nil {
			break
		}
		switch {
		case block.Type == "CERTIFICATE":
			cert = pem.EncodeToMemory(block)
		case block.Type == "PRIVATE KEY" || strings.HasSuffix(block.Type, " PRIVATE KEY"):
			if x509.IsEncryptedPEMBlock(block) {
				err = errors.New("private key PEM appears to be encrypted")
				break
			}
			key = pem.EncodeToMemory(block)
		default:
			err = fmt.Errorf("unrecognized PEM type: %q", block.Type)
		}
	}
	return
}

// StorePushCertHandler reads a PEM-encoded certificate and private
// key from the HTTP body and saves it to storage. This effectively
// enables us to do something like:
// "% cat push.pem push.key | curl -T - http://api.example.com/" to
// upload our push certs.
func StorePushCertHandler(storage storage.PushCertStore, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := ctxlog.Logger(r.Context(), logger)
		b, err := mdmhttp.ReadAllAndReplaceBody(r)
		if err != nil {
			logger.Info("msg", "reading body", "err", err)
			var toErr interface{ Timeout() bool }
			if errors.As(err, &toErr) && toErr.Timeout() {
				http.Error(w, http.StatusText(http.StatusRequestTimeout), http.StatusRequestTimeout)
				return
			}
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		certPEM, keyPEM, err := readPEMCertAndKey(b)
		if err == nil {
			// sanity check the provided cert and key to make sure they're usable as a pair.
			_, err = tls.X509KeyPair(certPEM, keyPEM)
		}
		var cert *x509.Certificate
		if err == nil {
			cert, err = cryptoutil.DecodePEMCertificate(certPEM)
		}
		var topic string
		if err == nil {
			topic, err = cryptoutil.TopicFromCert(cert)
		}
		if err == nil {
			err = storage.StorePushCert(r.Context(), certPEM, keyPEM)
		}
		output := &struct {
			Error    string    `json:"error,omitempty"`
			Topic    string    `json:"topic,omitempty"`
			NotAfter time.Time `json:"not_after,omitempty"`
		}{
			Topic: topic,
		}
		if cert != nil {
			output.NotAfter = cert.NotAfter
		}
		if err != nil {
			logger.Info("msg", "store push cert", "err", err)
			output.Error = err.Error()
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			logger.Debug("msg", "stored push cert", "topic", topic)
		}
		json, err := json.MarshalIndent(output, "", "\t")
		if err != nil {
			logger.Info("msg", "marshal json", "err", err)
		}
		w.Header().Set("Content-type", "application/json")
		_, err = w.Write(json)
		if err != nil {
			logger.Info("msg", "writing body", "err", err)
		}
	}
}
