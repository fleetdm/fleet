package pubsub

import (
	"context"
	"fmt"
	"sync"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

type inmemQueryResults struct {
	resultChannels map[uint]chan interface{}
	channelMutex   sync.Mutex
}

var _ fleet.QueryResultStore = &inmemQueryResults{}

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

func (im *inmemQueryResults) WriteResult(result fleet.DistributedQueryResult) error {
	channel := im.getChannel(result.DistributedQueryCampaignID)

	select {
	case channel <- result:
		// intentionally do nothing
	default:
		return noSubscriberError{fmt.Sprint(result.DistributedQueryCampaignID)}
	}

	return nil
}

func (im *inmemQueryResults) ReadChannel(ctx context.Context, campaign fleet.DistributedQueryCampaign) (<-chan interface{}, error) {
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
