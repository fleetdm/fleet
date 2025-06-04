// Package service retrieves push details from storage and sends MDM
// push notifications.
package service

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/storage"

	"github.com/micromdm/nanolib/log"
	"github.com/micromdm/nanolib/log/ctxlog"
)

type provider struct {
	provider   push.PushProvider
	staleToken string
}

// PushService uses PushStore to retrieve Push details for MDM enrollments
// and send APNs pushes using a PushProvider.
type PushService struct {
	store           storage.PushStore
	certStore       storage.PushCertStore
	providers       map[string]*provider
	providersMu     sync.RWMutex
	logger          log.Logger
	providerFactory push.PushProviderFactory
}

// NewPushService creates a new PushService.
func New(store storage.PushStore, certStore storage.PushCertStore, providerFactory push.PushProviderFactory, logger log.Logger) *PushService {
	return &PushService{
		logger:          logger,
		store:           store,
		certStore:       certStore,
		providers:       make(map[string]*provider),
		providerFactory: providerFactory,
	}
}

// getProvider returns a PushProvider if it exists and is not stale.
// Otherwise it creates a new PushProvider by retrieving the push certs.
func (s *PushService) getProvider(ctx context.Context, topic string) (push.PushProvider, error) {
	var (
		err   error
		stale bool
	)
	s.providersMu.RLock()
	prov := s.providers[topic]
	s.providersMu.RUnlock()
	if prov != nil && prov.provider != nil {
		stale, err = s.certStore.IsPushCertStale(ctx, topic, prov.staleToken)
		if err != nil {
			return nil, fmt.Errorf("checking push cert stale: %w", err)
		}
		if !stale {
			return prov.provider, nil
		}
	}
	cert, staleToken, err := s.certStore.RetrievePushCert(ctx, topic)
	if err != nil {
		return nil, fmt.Errorf("retrieving push cert for topic %q: %w", topic, err)
	}
	ctxlog.Logger(ctx, s.logger).Info(
		"msg", "retrieved push cert",
		"topic", topic,
	)
	newProvider, err := s.providerFactory.NewPushProvider(cert)
	if err != nil {
		return nil, fmt.Errorf("creating new push provider: %w", err)
	}
	prov = &provider{
		provider:   newProvider,
		staleToken: staleToken,
	}
	s.providersMu.Lock()
	s.providers[topic] = prov
	s.providersMu.Unlock()
	return prov.provider, nil
}

type pushFeedback struct {
	Responses map[string]*push.Response
	Err       error
	Topic     string
}

var ErrIdNotFound = errors.New("push data missing for id")

// push sends Push notifications to a push provider sychronously.
// pushInfos are mapped by push topic. The return maps push tokens
// (not IDs) to responses.
func (s *PushService) pushSingle(ctx context.Context, pushInfo *mdm.Push) (map[string]*push.Response, error) {
	if pushInfo == nil {
		return nil, errors.New("invalid push data")
	}
	prov, err := s.getProvider(ctx, pushInfo.Topic)
	if err != nil {
		return nil, err
	}
	return prov.Push(ctx, []*mdm.Push{pushInfo})
}

// pushMulti sends pushes to (potentially) multiple push providers
// asynchronously. The return maps push tokens (not IDs) to responses.
func (s *PushService) pushMulti(ctx context.Context, pushInfos []*mdm.Push) (map[string]*push.Response, error) {
	topicToPushInfos := make(map[string][]*mdm.Push)
	// split mdm.Pushs into topic separated map
	for _, pushInfo := range pushInfos {
		topicToPushInfos[pushInfo.Topic] = append(topicToPushInfos[pushInfo.Topic], pushInfo)
	}
	var finalErr error
	topicPushCt := 0
	feedbackChan := make(chan pushFeedback)
	for topic, pushInfos := range topicToPushInfos {
		prov, err := s.getProvider(ctx, topic)
		if err != nil {
			ctxlog.Logger(ctx, s.logger).Info(
				"msg", "get provider",
				"err", err,
			)
			finalErr = err
			continue
		}
		topicPushCt += 1
		go func(prov push.PushProvider, pushInfos []*mdm.Push, feedback chan<- pushFeedback, topic string) {
			resp, err := prov.Push(ctx, pushInfos)
			feedback <- pushFeedback{
				Responses: resp,
				Err:       err,
				Topic:     topic,
			}
		}(prov, pushInfos, feedbackChan, topic)
	}
	responses := make(map[string]*push.Response)
	for i := 0; i < topicPushCt; i++ {
		feedback := <-feedbackChan
		// merge feedback responses into main responses map
		for token, pushResp := range feedback.Responses {
			responses[token] = pushResp
		}
		if finalErr == nil && feedback.Err != nil {
			finalErr = fmt.Errorf("topic %s: %w", feedback.Topic, feedback.Err)
		}
	}
	close(feedbackChan)
	return responses, finalErr
}

// Push sends an APNs push notification to MDM enrollment id
func (s *PushService) Push(ctx context.Context, ids []string) (map[string]*push.Response, error) {
	idToPushInfo, err := s.store.RetrievePushInfo(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("push storage: %w", err)
	}
	idToResponse := make(map[string]*push.Response)

	// create mappings between tokens and enrollment IDs. Push providers
	// don't know about IDs and instead deal with Tokens as identifiers.
	tokenToId := make(map[string]string)
	pushInfos := make([]*mdm.Push, 0) // gather all pushInfos
	for _, id := range ids {
		if _, found := idToPushInfo[id]; found {
			pushInfo := idToPushInfo[id]
			pushInfos = append(pushInfos, pushInfo)
			// map token string back to id (Push Providers only know
			// of the push topic, not the identifier)
			tokenToId[pushInfo.Token.String()] = id
		} else {
			// populate a not found error for ids we requested but
			// storage did not return a result for
			idToResponse[id] = &push.Response{
				Id:  "",
				Err: ErrIdNotFound,
			}
		}
	}

	// perform actual pushes. we're dealing with maps keyed by token.
	var tokenToResponse map[string]*push.Response
	if len(pushInfos) == 1 {
		// some environments may heavily utilize individual pushes.
		// this justifies the special case and optimizes for it.
		tokenToResponse, err = s.pushSingle(ctx, pushInfos[0]) //nolint:gosec
	} else if len(pushInfos) > 1 {
		tokenToResponse, err = s.pushMulti(ctx, pushInfos)
	}

	// re-associate token responses with ids
	for token, resp := range tokenToResponse {
		id, ok := tokenToId[token]
		if !ok {
			ctxlog.Logger(ctx, s.logger).Info(
				"msg", "could not find id by token",
			)
			continue
		}
		idToResponse[id] = resp
	}

	return idToResponse, err
}
