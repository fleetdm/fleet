package file

import (
	"context"
	"errors"
	"os"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
)

// RetrievePushInfo retrieves APNs-related data for push notifications
func (s *FileStorage) RetrievePushInfo(_ context.Context, ids []string) (map[string]*mdm.Push, error) {
	pushInfos := make(map[string]*mdm.Push)
	for _, id := range ids {
		e := s.newEnrollment(id)
		tokenUpdate, err := e.readFile(TokenUpdateFilename)
		if errors.Is(err, os.ErrNotExist) {
			// TokenUpdate file missing could be a non-existent or
			// incomplete enrollment which should not trigger an error.
			continue
		} else if err != nil {
			return nil, err
		}
		msg, err := mdm.DecodeCheckin(tokenUpdate)
		if err != nil {
			return nil, err
		}
		message, ok := msg.(*mdm.TokenUpdate)
		if !ok {
			return nil, errors.New("saved TokenUpdate is not a TokenUpdate")
		}
		pushInfos[id] = &message.Push
	}
	return pushInfos, nil
}
