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
	converted        bool
	convertedErr     error
	recordBucketErr  error
	recordChangesErr error
	convertResults   []convertResult
	retireResults    []convertResult
	calls            []string
}

type convertResult struct {
	more bool
	err  error
}

func (f *fakeCVEStore) count(name string) int {
	var n int
	for _, c := range f.calls {
		if c == name {
			n++
		}
	}
	return n
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
	f.calls = append(f.calls, "RecordBucketData")
	return f.recordBucketErr
}

func (f *fakeCVEStore) RecordSCDChanges(context.Context, string, time.Time, map[string]*roaring.Bitmap) error {
	f.calls = append(f.calls, "RecordSCDChanges")
	return f.recordChangesErr
}

func (f *fakeCVEStore) ConvertSCDHistory(context.Context, string) (bool, error) {
	r := f.convertResults[min(f.count("ConvertSCDHistory"), len(f.convertResults)-1)]
	f.calls = append(f.calls, "ConvertSCDHistory")
	return r.more, r.err
}

func (f *fakeCVEStore) SCDHistoryConverted(context.Context, string) (bool, error) {
	f.calls = append(f.calls, "SCDHistoryConverted")
	return f.converted, f.convertedErr
}

func (f *fakeCVEStore) RetireSCDData(context.Context, string) (bool, error) {
	r := f.retireResults[min(f.count("RetireSCDData"), len(f.retireResults)-1)]
	f.calls = append(f.calls, "RetireSCDData")
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
			require.Equal(t, c.wantCalls, store.count("ConvertSCDHistory"))
			require.Equal(t, 1, store.count("RecordBucketData"), "the full copies are written until the history is converted")
			require.Zero(t, store.count("RetireSCDData"), "the full copies are kept until the history is converted")
		})
	}
}

func TestCVEDatasetCollectRetiresFullCopies(t *testing.T) {
	defer func(b time.Duration) { cveRetireBudget = b }(cveRetireBudget)
	boom := errors.New("boom")

	cases := []struct {
		name             string
		budget           time.Duration
		recordChangesErr error
		results          []convertResult
		wantCalls        []string
		wantErr          error
	}{
		{name: "batches run until none remains", budget: time.Hour,
			results:   []convertResult{{more: true}, {more: true}, {more: false}},
			wantCalls: []string{"SCDHistoryConverted", "RecordSCDChanges", "RetireSCDData", "RetireSCDData", "RetireSCDData"}},
		{name: "the budget ends the loop after at least one batch", budget: 0,
			results:   []convertResult{{more: true}},
			wantCalls: []string{"SCDHistoryConverted", "RecordSCDChanges", "RetireSCDData"}},
		{name: "a delete error is returned", budget: time.Hour,
			results:   []convertResult{{err: boom}},
			wantCalls: []string{"SCDHistoryConverted", "RecordSCDChanges", "RetireSCDData"}, wantErr: boom},
		{name: "a busy change storage skips the delete", budget: time.Hour, recordChangesErr: api.ErrChangeStorageBusy,
			results:   []convertResult{{more: true}},
			wantCalls: []string{"SCDHistoryConverted", "RecordSCDChanges"}},
		{name: "a change storage error skips the delete", budget: time.Hour, recordChangesErr: boom,
			results:   []convertResult{{more: true}},
			wantCalls: []string{"SCDHistoryConverted", "RecordSCDChanges"}, wantErr: boom},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cveRetireBudget = c.budget
			store := &fakeCVEStore{converted: true, recordChangesErr: c.recordChangesErr, retireResults: c.results}
			err := (&CVEDataset{}).Collect(t.Context(), store, time.Now(), nil)
			if c.wantErr != nil {
				require.ErrorIs(t, err, c.wantErr)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, c.wantCalls, store.calls)
		})
	}

	t.Run("a failed conversion lookup writes nothing", func(t *testing.T) {
		store := &fakeCVEStore{convertedErr: boom}
		err := (&CVEDataset{}).Collect(t.Context(), store, time.Now(), nil)
		require.ErrorIs(t, err, boom)
		require.Equal(t, []string{"SCDHistoryConverted"}, store.calls)
	})
}
