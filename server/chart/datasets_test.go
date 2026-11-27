package chart

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/RoaringBitmap/roaring"
	"github.com/fleetdm/fleet/v4/server/chart/api"
	"github.com/stretchr/testify/require"
)

type fakeCVEStore struct {
	recordChangesErr error
	convertResults   []convertResult
	convertCalls     int
}

type convertResult struct {
	more bool
	err  error
}

func (f *fakeCVEStore) FindOnlineHostIDs(context.Context, time.Time, []uint) ([]uint, error) {
	return nil, nil
}

func (f *fakeCVEStore) AffectedHostIDsByCVE(context.Context, []uint, []string) (map[string]*roaring.Bitmap, error) {
	return map[string]*roaring.Bitmap{"CVE-1": roaring.BitmapOf(1)}, nil
}

func (f *fakeCVEStore) CollectibleCVEs(context.Context) ([]string, error) {
	return []string{"CVE-1"}, nil
}

func (f *fakeCVEStore) RecordBucketData(context.Context, string, time.Time, time.Duration, api.SampleStrategy, map[string]*roaring.Bitmap) error {
	return nil
}

func (f *fakeCVEStore) RecordSCDChanges(context.Context, string, time.Time, map[string]*roaring.Bitmap) error {
	return f.recordChangesErr
}

func (f *fakeCVEStore) ConvertSCDHistory(context.Context, string) (bool, error) {
	r := f.convertResults[min(f.convertCalls, len(f.convertResults)-1)]
	f.convertCalls++
	return r.more, r.err
}

func TestCVEDatasetCollectConvertsHistory(t *testing.T) {
	defer func(b time.Duration) { cveConvertBudget = b }(cveConvertBudget)
	boom := errors.New("boom")

	cases := []struct {
		name             string
		budget           time.Duration
		recordChangesErr error
		results          []convertResult
		wantCalls        int
		wantErr          error
	}{
		{name: "steps run until none remains", budget: time.Hour,
			results: []convertResult{{more: true}, {more: true}, {more: false}}, wantCalls: 3},
		{name: "the budget ends the loop after at least one step", budget: 0,
			results: []convertResult{{more: true}}, wantCalls: 1},
		{name: "a busy conversion skips the tick", budget: time.Hour,
			results: []convertResult{{more: true}, {err: api.ErrChangeStorageBusy}}, wantCalls: 2},
		{name: "a conversion error is returned", budget: time.Hour,
			results: []convertResult{{err: boom}}, wantCalls: 1, wantErr: boom},
		{name: "a busy change storage skips the conversion", budget: time.Hour, recordChangesErr: api.ErrChangeStorageBusy,
			results: []convertResult{{more: true}}, wantCalls: 0},
		{name: "a change storage error skips the conversion", budget: time.Hour, recordChangesErr: boom,
			results: []convertResult{{more: true}}, wantCalls: 0, wantErr: boom},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cveConvertBudget = c.budget
			store := &fakeCVEStore{recordChangesErr: c.recordChangesErr, convertResults: c.results}
			err := (&CVEDataset{}).Collect(t.Context(), store, time.Now(), nil)
			if c.wantErr != nil {
				require.ErrorIs(t, err, c.wantErr)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, c.wantCalls, store.convertCalls)
		})
	}
}
