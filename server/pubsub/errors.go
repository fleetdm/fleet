// Package pubsub implements pub/sub interfaces defined in package fleet.
package pubsub

// Error defines the interface of errors specific to the pubsub package
type Error interface {
	error
	// NoSubscriber returns true if the error occurred because there are no
	// subscribers on the channel
	NoSubscriber() bool
}

// NoSubscriberError can be returned when channel operations fail because there
// are no subscribers. Its NoSubscriber() method always returns true.
type noSubscriberError struct {
	Channel string
}

func (e noSubscriberError) Error() string {
	return "no subscriber for channel " + e.Channel
}

func (e noSubscriberError) NoSubscriber() bool {
	return true
}
