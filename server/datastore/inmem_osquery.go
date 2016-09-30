package datastore

import (
	"errors"
	"strconv"
	"time"

	"github.com/kolide/kolide-ose/server/kolide"
)

func (orm *inmem) NewLabel(label *kolide.Label) (*kolide.Label, error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	newLabel := *label

	for _, l := range orm.labels {
		if l.Name == label.Name {
			return nil, ErrExists
		}
	}

	newLabel.ID = uint(len(orm.labels) + 1)
	orm.labels[newLabel.ID] = &newLabel

	return &newLabel, nil
}

func (orm *inmem) LabelsForHost(host *kolide.Host) ([]kolide.Label, error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	// First get IDs of label executions for the host
	resLabels := []kolide.Label{}
	for _, lqe := range orm.labelQueryExecutions {
		if lqe.HostID == host.ID && lqe.Matches {
			if label := orm.labels[lqe.LabelID]; label != nil {
				resLabels = append(resLabels, *label)
			}
		}
	}

	return resLabels, nil
}

func (orm *inmem) LabelQueriesForHost(host *kolide.Host, cutoff time.Time) (map[string]string, error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	// Get post-cutoff executions for host
	execedQueryIDs := map[uint]uint{} // Map queryID -> labelID
	for _, lqe := range orm.labelQueryExecutions {
		if lqe.HostID == host.ID && (lqe.UpdatedAt == cutoff || lqe.UpdatedAt.After(cutoff)) {
			label := orm.labels[lqe.LabelID]
			execedQueryIDs[label.QueryID] = label.ID
		}
	}

	queryToLabel := map[uint]uint{} // Map queryID -> labelID
	for _, label := range orm.labels {
		queryToLabel[label.QueryID] = label.ID
	}

	resQueries := map[string]string{}
	for _, query := range orm.queries {
		_, execed := execedQueryIDs[query.ID]
		labelID := queryToLabel[query.ID]
		if query.Platform == host.Platform && !execed {
			resQueries[strconv.Itoa(int(labelID))] = query.Query
		}
	}

	return resQueries, nil
}

func (orm *inmem) getLabelByIDString(id string) (*kolide.Label, error) {
	labelID, err := strconv.Atoi(id)
	if err != nil {
		return nil, errors.New("non-int label ID")
	}

	label, ok := orm.labels[uint(labelID)]
	if !ok {
		return nil, errors.New("label ID not found: " + string(labelID))
	}

	return label, nil
}

func (orm *inmem) RecordLabelQueryExecutions(host *kolide.Host, results map[string]bool, t time.Time) error {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	// Record executions
	for strLabelID, matches := range results {
		label, err := orm.getLabelByIDString(strLabelID)
		if err != nil {
			return err
		}

		updated := false
		for _, lqe := range orm.labelQueryExecutions {
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
				ID:        uint(len(orm.labelQueryExecutions) + 1),
				HostID:    host.ID,
				LabelID:   label.ID,
				UpdatedAt: t,
				Matches:   matches,
			}
			orm.labelQueryExecutions[lqe.ID] = &lqe
		}
	}

	return nil
}
