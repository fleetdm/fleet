package pubsub

import (
	"errors"
	"sync"

	"golang.org/x/net/context"

	"github.com/kolide/kolide-ose/server/kolide"
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
		return errors.New("no subscribers for channel")
	}

	select {
	case channel <- result:
		// intentionally do nothing
	default:
		return errors.New("no subscribers for channel")
	}

	return nil
}

func (im *inmemQueryResults) ReadChannel(ctx context.Context, query kolide.DistributedQueryCampaign) (<-chan interface{}, error) {
	channel := im.getChannel(query.ID)
	go func() {
		<-ctx.Done()
		close(channel)
		im.channelMutex.Lock()
		delete(im.resultChannels, query.ID)
		im.channelMutex.Unlock()
	}()
	return channel, nil
}
