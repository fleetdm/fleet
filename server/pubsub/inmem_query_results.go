package pubsub

import (
	"context"
	"strconv"
	"sync"

	"github.com/fleetdm/fleet/server/kolide"
)

type inmemQueryResults struct {
	resultChannels map[uint]chan interface{}
	channelMutex   sync.Mutex
}

var _ kolide.QueryResultStore = &inmemQueryResults{}

// NewInmemQueryResults initializes a new in-memory implementation of the
// QueryResultStore interface.
func NewInmemQueryResults() *inmemQueryResults {
	return &inmemQueryResults{resultChannels: map[uint]chan interface{}{}}
}

func (im *inmemQueryResults) getChannel(id uint) chan interface{} {
	im.channelMutex.Lock()
	defer im.channelMutex.Unlock()

	channel, ok := im.resultChannels[id]
	if !ok {
		channel = make(chan interface{})
		im.resultChannels[id] = channel
	}
	return channel
}

func (im *inmemQueryResults) WriteResult(result kolide.DistributedQueryResult) error {
	channel, ok := im.resultChannels[result.DistributedQueryCampaignID]
	if !ok {
		return noSubscriberError{strconv.Itoa(int(result.DistributedQueryCampaignID))}
	}

	select {
	case channel <- result:
		// intentionally do nothing
	default:
		return noSubscriberError{strconv.Itoa(int(result.DistributedQueryCampaignID))}
	}

	return nil
}

func (im *inmemQueryResults) ReadChannel(ctx context.Context, campaign kolide.DistributedQueryCampaign) (<-chan interface{}, error) {
	channel := im.getChannel(campaign.ID)
	go func() {
		<-ctx.Done()
		close(channel)
		im.channelMutex.Lock()
		delete(im.resultChannels, campaign.ID)
		im.channelMutex.Unlock()
	}()
	return channel, nil
}

func (im *inmemQueryResults) HealthCheck() error {
	return nil
}
