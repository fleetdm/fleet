package file

import (
	"context"
	"errors"
	"os"
	"path"
	"strings"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
)

const (
	subNotNow   = "QueueNotNow"
	subQueue    = "Queue"
	subDone     = "QueueDone"
	subInactive = "QueueInactive"
)

type queue struct {
	e   *enrollment
	sub string // subdirectory for this queue
}

func (e *enrollment) newQueue(sub string) *queue {
	if sub == "" {
		sub = subQueue
	}
	return &queue{e: e, sub: sub}
}

func (q *queue) dir() string {
	return path.Join(q.e.dir(), q.sub)
}

func (q *queue) mkdir() error {
	return os.MkdirAll(q.dir(), 0755)
}

func (q *queue) enqueue(uuid string, raw []byte) error {
	err := q.mkdir()
	if err != nil {
		return err
	}
	return os.WriteFile( //nolint:gosec
		path.Join(q.dir(), uuid+".plist"),
		raw,
		0755,
	)
}

func (q *queue) exists(uuid string) (bool, error) {
	if _, err := os.Stat(path.Join(q.dir(), uuid+".plist")); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (q *queue) move(uuid string, dest *queue) error {
	err := dest.mkdir()
	if err != nil {
		return err
	}
	return os.Rename(
		path.Join(q.dir(), uuid+".plist"),
		path.Join(dest.dir(), uuid+".plist"),
	)
}

func (q *queue) removeResults(uuid string) error {
	return os.Remove(path.Join(q.dir(), uuid+".result.plist"))
}

func (q *queue) writeResults(uuid string, raw []byte) error {
	return os.WriteFile( //nolint:gosec
		path.Join(q.dir(), uuid+".result.plist"),
		raw,
		0755,
	)
}

func (q *queue) getNext() (*mdm.Command, error) {
	entries, err := os.ReadDir(q.dir())
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".plist") {
			continue
		}
		raw, err := os.ReadFile(path.Join(q.dir(), entry.Name()))
		if err != nil {
			return nil, err
		}
		return mdm.DecodeCommand(raw)
	}
	return nil, nil
}

// EnqueueCommand writes the command to disk in the queue directory
func (s *FileStorage) EnqueueCommand(_ context.Context, ids []string, command *mdm.CommandWithSubtype) (map[string]error,
	error) {
	idErrs := make(map[string]error)
	for _, id := range ids {
		e := s.newEnrollment(id)
		q := e.newQueue(subQueue)
		if err := q.enqueue(command.CommandUUID, command.Raw); err != nil {
			idErrs[id] = err
		}
	}
	return idErrs, nil
}

// StoreCommandReport moves commands to different queues (like NotNow)
func (s *FileStorage) StoreCommandReport(r *mdm.Request, report *mdm.CommandResults) error {
	if report.Status == "Idle" {
		return nil
	}
	e := s.newEnrollment(r.ID)
	src := e.newQueue(subQueue)
	qExists, err := src.exists(report.CommandUUID)
	if err != nil {
		return err
	}
	nnq := e.newQueue(subNotNow)
	var nnqExists bool
	if !qExists {
		nnqExists, err = nnq.exists(report.CommandUUID)
		if err != nil {
			return err
		}
		if nnqExists {
			src = nnq

		}
	}
	dest := e.newQueue(subDone)
	if report.Status == "NotNow" {
		dest = e.newQueue(subNotNow)
	}
	err = src.move(report.CommandUUID, dest)
	if err != nil {
		return err
	}
	if nnqExists {
		if err := nnq.removeResults(report.CommandUUID); err != nil {
			return err
		}
	}
	return dest.writeResults(report.CommandUUID, report.Raw)
}

// RetrieveNextCommand gets the next command from the queue while minding NotNow status
func (s *FileStorage) RetrieveNextCommand(r *mdm.Request, skipNotNow bool) (*mdm.CommandWithSubtype, error) {
	e := s.newEnrollment(r.ID)
	var q *queue
	if !skipNotNow {
		q = e.newQueue(subNotNow)
		raw, err := q.getNext()
		if err != nil {
			return nil, err
		}
		if raw != nil {
			return &mdm.CommandWithSubtype{Command: *raw}, nil
		}
	}
	q = e.newQueue(subQueue)
	raw, err := q.getNext()
	if err != nil {
		return nil, err
	}
	if raw != nil {
		return &mdm.CommandWithSubtype{Command: *raw}, nil
	}
	return nil, nil
}

func (s *FileStorage) ClearQueue(r *mdm.Request) error {
	if r.ParentID != "" {
		return errors.New("can only clear a device channel queue")
	}
	// assemble list of IDs for which to clear the queue
	e := s.newEnrollment(r.ID)
	clearIds := e.listSubEnrollments()
	clearIds = append(clearIds, r.ID)
	// clear the queue for all of the ids
	for _, id := range clearIds {
		e := s.newEnrollment(id)
		dest := e.newQueue(subInactive)
		for _, q := range []*queue{e.newQueue(subQueue), e.newQueue(subNotNow)} {
			raw, err := q.getNext()
			for raw != nil && err == nil {
				err = q.move(raw.CommandUUID, dest)
				if err != nil {
					return err
				}
				raw, err = q.getNext()
			}
			if err != nil {
				return err
			}
		}
	}
	return nil
}
