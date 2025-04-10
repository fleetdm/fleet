// Pacakge buford adapts the buford APNs push package to the PushProvider and
// PushProviderFactory interfaces.
package buford

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"time"

	bufordpush "github.com/RobotsAndPencils/buford/push"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push"
)

// NewClient describes a callback for setting up an HTTP client for Push notifications.
type NewClient func(*tls.Certificate) (*http.Client, error)

// bufordFactory instantiates new buford Services to satisfy the PushProviderFactory interface.
type bufordFactory struct {
	workers           uint
	expiration        time.Duration
	newClientCallback NewClient
}

type Option func(*bufordFactory)

// WithWorkers sets how many worker goroutines to use when sending
// multiple push notifications.
func WithWorkers(workers uint) Option {
	return func(f *bufordFactory) {
		f.workers = workers
	}
}

// WithExpiration sets the APNs expiration time for the push notifications.
func WithExpiration(expiration time.Duration) Option {
	return func(f *bufordFactory) {
		f.expiration = expiration
	}
}

// WithNewClient sets a callback to setup an HTTP client for each
// new Push provider.
func WithNewClient(newClientCallback NewClient) Option {
	return func(f *bufordFactory) {
		f.newClientCallback = newClientCallback
	}
}

// NewPushProviderFactory creates a new instance that can spawn buford Services
func NewPushProviderFactory(opts ...Option) *bufordFactory {
	factory := &bufordFactory{
		workers: 5,
	}
	for _, opt := range opts {
		opt(factory)
	}
	return factory
}

// NewPushProvider generates a new PushProvider given a tls keypair
func (f *bufordFactory) NewPushProvider(cert *tls.Certificate) (push.PushProvider, error) {
	var client *http.Client
	var err error
	if f.newClientCallback == nil {
		client, err = bufordpush.NewClient(*cert)
	} else {
		client, err = f.newClientCallback(cert)
	}
	if err != nil {
		return nil, err
	}
	prov := &bufordPushProvider{
		service:    bufordpush.NewService(client, bufordpush.Production),
		expiration: f.expiration,
		workers:    f.workers,
	}
	return prov, err
}

// bufordPushProvider wraps a buford Service to satisfy the PushProvider interface.
type bufordPushProvider struct {
	service    *bufordpush.Service
	expiration time.Duration
	workers    uint
}

func assemblePushData(magic string, exp time.Duration) (payload []byte, hdr *bufordpush.Headers) {
	payload = []byte(`{"mdm":"` + magic + `"}`)
	if exp > 0 {
		expiration := time.Now().Add(exp)
		hdr = &bufordpush.Headers{Expiration: expiration}
	}
	return
}

func (c *bufordPushProvider) pushSingle(pushInfo *mdm.Push) *push.Response {
	resp := new(push.Response)
	payload, headers := assemblePushData(pushInfo.PushMagic, c.expiration)
	resp.Id, resp.Err = c.service.Push(pushInfo.Token.String(), headers, payload)
	return resp
}

func (c *bufordPushProvider) pushMulti(pushInfos []*mdm.Push) map[string]*push.Response {
	workers := uint(len(pushInfos))
	if workers > c.workers {
		workers = c.workers
	}
	queue := bufordpush.NewQueue(c.service, workers)
	defer queue.Close()
	for _, push := range pushInfos {
		payload, headers := assemblePushData(push.PushMagic, c.expiration)
		go queue.Push(push.Token.String(), headers, payload)
	}
	responses := make(map[string]*push.Response)
	for range pushInfos {
		bufordResp := <-queue.Responses
		responses[bufordResp.DeviceToken] = &push.Response{
			Id:  bufordResp.ID,
			Err: bufordResp.Err,
		}
	}
	return responses
}

// Push sends 'raw' MDM APNs push notifications to service in c.
func (c *bufordPushProvider) Push(_ context.Context, pushInfos []*mdm.Push) (map[string]*push.Response, error) {
	if len(pushInfos) < 1 {
		return nil, errors.New("no push data provided")
	}
	// some environments may heavily utilize individual pushes.
	// this justifies the special case and optimizes for it.
	if len(pushInfos) == 1 {
		responses := make(map[string]*push.Response)
		responses[pushInfos[0].Token.String()] = c.pushSingle(pushInfos[0])
		return responses, nil
	}
	return c.pushMulti(pushInfos), nil
}
