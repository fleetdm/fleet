package main

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/facebookincubator/nvdtools/cpedict"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMaybeSendStatistics(t *testing.T) {
	ds := new(mock.Store)

	requestBody := ""

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestBodyBytes, err := ioutil.ReadAll(r.Body)
		require.NoError(t, err)
		requestBody = string(requestBodyBytes)
	}))
	defer ts.Close()

	ds.AppConfigFunc = func() (*fleet.AppConfig, error) {
		return &fleet.AppConfig{EnableAnalytics: true}, nil
	}

	ds.ShouldSendStatisticsFunc = func(frequency time.Duration) (fleet.StatisticsPayload, bool, error) {
		return fleet.StatisticsPayload{
			AnonymousIdentifier: "ident",
			FleetVersion:        "1.2.3",
			NumHostsEnrolled:    999,
		}, true, nil
	}
	recorded := false
	ds.RecordStatisticsSentFunc = func() error {
		recorded = true
		return nil
	}

	err := trySendStatistics(ds, fleet.StatisticsFrequency, ts.URL)
	require.NoError(t, err)
	assert.True(t, recorded)
	assert.Equal(t, `{"anonymousIdentifier":"ident","fleetVersion":"1.2.3","numHostsEnrolled":999}`, requestBody)
}

func TestMaybeSendStatisticsSkipsSendingIfNotNeeded(t *testing.T) {
	ds := new(mock.Store)

	called := false

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer ts.Close()

	ds.AppConfigFunc = func() (*fleet.AppConfig, error) {
		return &fleet.AppConfig{EnableAnalytics: true}, nil
	}

	ds.ShouldSendStatisticsFunc = func(frequency time.Duration) (fleet.StatisticsPayload, bool, error) {
		return fleet.StatisticsPayload{}, false, nil
	}
	recorded := false
	ds.RecordStatisticsSentFunc = func() error {
		recorded = true
		return nil
	}

	err := trySendStatistics(ds, fleet.StatisticsFrequency, ts.URL)
	require.NoError(t, err)
	assert.False(t, recorded)
	assert.False(t, called)
}

func TestMaybeSendStatisticsSkipsIfNotConfigured(t *testing.T) {
	ds := new(mock.Store)

	called := false

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer ts.Close()

	ds.AppConfigFunc = func() (*fleet.AppConfig, error) {
		return &fleet.AppConfig{EnableAnalytics: false}, nil
	}

	err := trySendStatistics(ds, fleet.StatisticsFrequency, ts.URL)
	require.NoError(t, err)
	assert.False(t, called)
}

type fakeSoftwareIterator struct {
	index     int
	softwares []*fleet.Software
}

func (f *fakeSoftwareIterator) Next() bool {
	return f.index < len(f.softwares)
}

func (f *fakeSoftwareIterator) Value() (*fleet.Software, error) {
	s := f.softwares[f.index]
	f.index++
	return s, nil
}

func (f *fakeSoftwareIterator) Err() error { return nil }

func TestTranslateSoftwareToCPE(t *testing.T) {
	tempDir := os.TempDir()

	ds := new(mock.Store)
	ds.AppConfigFunc = func() (*fleet.AppConfig, error) {
		return &fleet.AppConfig{VulnerabilityDatabasesPath: &tempDir}, nil
	}

	var cpes []string

	ds.AddCPEForSoftwareFunc = func(software fleet.Software, cpe string) error {
		cpes = append(cpes, cpe)
		return nil
	}

	ds.AllSoftwareIteratorFunc = func() (fleet.SoftwareIterator, error) {
		return &fakeSoftwareIterator{
			softwares: []*fleet.Software{
				{
					ID:      1,
					Name:    "Product",
					Version: "1.2.3",
					Source:  "apps",
				},
				{
					ID:      2,
					Name:    "Product2",
					Version: "0.3",
					Source:  "apps",
				},
			},
		}, nil
	}

	items, err := cpedict.Decode(strings.NewReader(vulnerabilities.XmlCPETestDict))
	require.NoError(t, err)

	dbPath := path.Join(tempDir, "cpe.sqlite")
	err = vulnerabilities.GenerateCPEDB(dbPath, items)
	require.NoError(t, err)

	err = translateSoftwareToCPE(ds)
	require.NoError(t, err)
	assert.Equal(t, []string{
		"cpe:2.3:a:vendor:product:1.2.3:*:*:*:*:macos:*:*",
		"cpe:2.3:a:vendor2:product4:999:*:*:*:*:macos:*:*",
	}, cpes)
}
