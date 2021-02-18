package inmem

import (
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/fleetdm/fleet/server/kolide"
	"github.com/patrickmn/sortutil"
)

func (d *Datastore) NewLabel(label *kolide.Label, opts ...kolide.OptionalArg) (*kolide.Label, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	newLabel := *label

	for _, l := range d.labels {
		if l.Name == label.Name {
			return nil, alreadyExists("Label", l.ID)
		}
	}

	newLabel.ID = d.nextID(label)
	d.labels[newLabel.ID] = &newLabel

	return &newLabel, nil
}

func (d *Datastore) ListLabelsForHost(hid uint) ([]kolide.Label, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	// First get IDs of label executions for the host
	resLabels := []kolide.Label{}

	for _, lqe := range d.labelQueryExecutions {
		if lqe.HostID == hid && lqe.Matches {
			if label := d.labels[lqe.LabelID]; label != nil {
				resLabels = append(resLabels, *label)
			}
		}
	}

	return resLabels, nil
}

func (d *Datastore) LabelQueriesForHost(host *kolide.Host, cutoff time.Time) (map[string]string, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	// Get post-cutoff executions for host
	execedIDs := map[uint]bool{}

	for _, lqe := range d.labelQueryExecutions {
		if lqe.HostID == host.ID && (lqe.UpdatedAt == cutoff || lqe.UpdatedAt.After(cutoff)) {
			execedIDs[lqe.LabelID] = true
		}
	}

	queries := map[string]string{}
	for _, label := range d.labels {
		if (label.Platform == "" || strings.Contains(label.Platform, host.Platform)) && !execedIDs[label.ID] {
			queries[strconv.Itoa(int(label.ID))] = label.Query
		}
	}

	return queries, nil
}

func (d *Datastore) RecordLabelQueryExecutions(host *kolide.Host, results map[uint]bool, t time.Time) error {
	// Record executions
	for labelID, matches := range results {
		label, ok := d.labels[labelID]
		if !ok {
			return notFound("Label").WithID(labelID)
		}

		updated := false
		d.mtx.Lock()
		for _, lqe := range d.labelQueryExecutions {
			if lqe.LabelID == label.ID && lqe.HostID == host.ID {
				// Update existing execution values
				lqe.UpdatedAt = t
				lqe.Matches = matches
				updated = true
				break
			}
		}

		if !updated {
			// Create new execution
			lqe := kolide.LabelQueryExecution{
				HostID:    host.ID,
				LabelID:   label.ID,
				UpdatedAt: t,
				Matches:   matches,
			}
			lqe.ID = d.nextID(lqe)
			d.labelQueryExecutions[lqe.ID] = &lqe
		}
		d.mtx.Unlock()
	}

	return nil
}

func (d *Datastore) Label(lid uint) (*kolide.Label, error) {
	d.mtx.Lock()
	label, ok := d.labels[lid]
	d.mtx.Unlock()

	if !ok {
		return nil, errors.New("Label not found")
	}
	return label, nil
}

func (d *Datastore) ListLabels(opt kolide.ListOptions) ([]*kolide.Label, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	// We need to sort by keys to provide reliable ordering
	keys := []int{}

	for k := range d.labels {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)

	labels := []*kolide.Label{}
	for _, k := range keys {
		labels = append(labels, d.labels[uint(k)])
	}

	// Apply ordering
	if opt.OrderKey != "" {
		var fields = map[string]string{
			"id":         "ID",
			"created_at": "CreatedAt",
			"updated_at": "UpdatedAt",
			"name":       "Name",
		}
		if err := sortResults(labels, opt, fields); err != nil {
			return nil, err
		}
	}

	// Apply limit/offset
	low, high := d.getLimitOffsetSliceBounds(opt, len(labels))
	labels = labels[low:high]

	return labels, nil
}

func (d *Datastore) SearchLabels(query string, omit ...uint) ([]kolide.Label, error) {
	omitLookup := map[uint]bool{}
	for _, o := range omit {
		omitLookup[o] = true
	}

	var results []kolide.Label

	d.mtx.Lock()
	defer d.mtx.Unlock()

	for _, l := range d.labels {
		if len(results) == 10 {
			break
		}

		if (strings.Contains(l.Name, query) || l.Name == "All Hosts") && !omitLookup[l.ID] {
			results = append(results, *l)
			continue
		}
	}

	sortutil.AscByField(results, "ID")
	return results, nil
}

func (d *Datastore) ListHostsInLabel(lid uint, opt kolide.HostListOptions) ([]kolide.Host, error) {
	var hosts []kolide.Host

	d.mtx.Lock()
	defer d.mtx.Unlock()

	for _, lqe := range d.labelQueryExecutions {
		if lqe.LabelID == lid && lqe.Matches {
			hosts = append(hosts, *d.hosts[lqe.HostID])
		}
	}

	return hosts, nil
}

func (d *Datastore) ListUniqueHostsInLabels(labels []uint) ([]kolide.Host, error) {
	var hosts []kolide.Host

	labelSet := map[uint]bool{}
	hostSet := map[uint]bool{}

	for _, label := range labels {
		labelSet[label] = true
	}

	d.mtx.Lock()
	defer d.mtx.Unlock()

	for _, lqe := range d.labelQueryExecutions {
		if labelSet[lqe.LabelID] && lqe.Matches {
			if !hostSet[lqe.HostID] {
				hosts = append(hosts, *d.hosts[lqe.HostID])
				hostSet[lqe.HostID] = true
			}
		}
	}

	return hosts, nil
}

func (d *Datastore) SaveLabel(label *kolide.Label) (*kolide.Label, error) {
	panic("inmem is being deprecated")
}
