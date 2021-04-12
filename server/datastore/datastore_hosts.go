package datastore

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/server/kolide"
	"github.com/fleetdm/fleet/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var enrollTests = []struct {
	uuid, hostname, platform, nodeKey string
}{
	0: {uuid: "6D14C88F-8ECF-48D5-9197-777647BF6B26",
		hostname: "web.kolide.co",
		platform: "linux",
		nodeKey:  "key0",
	},
	1: {uuid: "B998C0EB-38CE-43B1-A743-FBD7A5C9513B",
		hostname: "mail.kolide.co",
		platform: "linux",
		nodeKey:  "key1",
	},
	2: {uuid: "008F0688-5311-4C59-86EE-00C2D6FC3EC2",
		hostname: "home.kolide.co",
		platform: "darwin",
		nodeKey:  "key2",
	},
	3: {uuid: "uuid123",
		hostname: "fakehostname",
		platform: "darwin",
		nodeKey:  "key3",
	},
}

func testSaveHosts(t *testing.T, ds kolide.Datastore) {
	host, err := ds.NewHost(&kolide.Host{
		DetailUpdateTime: time.Now(),
		LabelUpdateTime:  time.Now(),
		SeenTime:         time.Now(),
		NodeKey:          "1",
		UUID:             "1",
		HostName:         "foo.local",
		PrimaryIP:        "192.168.1.1",
		PrimaryMac:       "30-65-EC-6F-C4-58",
	})
	require.NoError(t, err)
	require.NotNil(t, host)

	host.HostName = "bar.local"
	err = ds.SaveHost(host)
	require.Nil(t, err)

	host, err = ds.Host(host.ID)
	require.Nil(t, err)
	assert.Equal(t, "bar.local", host.HostName)
	assert.Equal(t, "192.168.1.1", host.PrimaryIP)
	assert.Equal(t, "30-65-EC-6F-C4-58", host.PrimaryMac)

	additionalJSON := json.RawMessage(`{"foobar": "bim"}`)
	host.Additional = &additionalJSON

	err = ds.SaveHost(host)
	require.Nil(t, err)

	host, err = ds.Host(host.ID)
	require.Nil(t, err)
	require.NotNil(t, host)
	require.NotNil(t, host.Additional)
	assert.Equal(t, additionalJSON, *host.Additional)

	err = ds.SaveHost(host)
	require.Nil(t, err)

	host, err = ds.Host(host.ID)
	require.Nil(t, err)
	require.NotNil(t, host)

	err = ds.DeleteHost(host.ID)
	assert.Nil(t, err)

	host, err = ds.Host(host.ID)
	assert.NotNil(t, err)
	assert.Nil(t, host)
}

func testDeleteHost(t *testing.T, ds kolide.Datastore) {
	host, err := ds.NewHost(&kolide.Host{
		DetailUpdateTime: time.Now(),
		LabelUpdateTime:  time.Now(),
		SeenTime:         time.Now(),
		NodeKey:          "1",
		UUID:             "1",
		HostName:         "foo.local",
	})
	require.Nil(t, err)
	require.NotNil(t, host)

	err = ds.DeleteHost(host.ID)
	assert.Nil(t, err)

	host, err = ds.Host(host.ID)
	assert.NotNil(t, err)
}

func testListHosts(t *testing.T, ds kolide.Datastore) {
	hosts := []*kolide.Host{}
	for i := 0; i < 10; i++ {
		host, err := ds.NewHost(&kolide.Host{
			DetailUpdateTime: time.Now(),
			LabelUpdateTime:  time.Now(),
			SeenTime:         time.Now(),
			OsqueryHostID:    strconv.Itoa(i),
			NodeKey:          fmt.Sprintf("%d", i),
			UUID:             fmt.Sprintf("%d", i),
			HostName:         fmt.Sprintf("foo.local%d", i),
		})
		assert.Nil(t, err)
		if err != nil {
			return
		}
		hosts = append(hosts, host)
	}

	hosts2, err := ds.ListHosts(kolide.HostListOptions{})
	require.Nil(t, err)
	assert.Equal(t, len(hosts), len(hosts2))

	// Test with logic for only a few hosts
	hosts2, err = ds.ListHosts(kolide.HostListOptions{ListOptions: kolide.ListOptions{PerPage: 4, Page: 0}})
	require.Nil(t, err)
	assert.Equal(t, 4, len(hosts2))

	err = ds.DeleteHost(hosts[0].ID)
	require.Nil(t, err)
	hosts2, err = ds.ListHosts(kolide.HostListOptions{})
	require.Nil(t, err)
	assert.Equal(t, len(hosts)-1, len(hosts2))

	hosts, err = ds.ListHosts(kolide.HostListOptions{})
	require.Nil(t, err)
	require.Equal(t, len(hosts2), len(hosts))

	err = ds.SaveHost(hosts[0])
	require.Nil(t, err)
	hosts2, err = ds.ListHosts(kolide.HostListOptions{})
	require.Nil(t, err)
	require.Equal(t, hosts[0].ID, hosts2[0].ID)
}

func testListHostsFilterAdditional(t *testing.T, ds kolide.Datastore) {
	h, err := ds.NewHost(&kolide.Host{
		DetailUpdateTime: time.Now(),
		LabelUpdateTime:  time.Now(),
		SeenTime:         time.Now(),
		OsqueryHostID:    "foobar",
		NodeKey:          "nodekey",
		UUID:             "uuid",
		HostName:         "foobar.local",
	})
	require.Nil(t, err)

	// Add additional
	additional := json.RawMessage(`{"field1": "v1", "field2": "v2"}`)
	h.Additional = &additional
	err = ds.SaveHost(h)
	require.Nil(t, err)

	additional = json.RawMessage(`{"field1": "v1", "field2": "v2"}`)
	hosts, err := ds.ListHosts(kolide.HostListOptions{})
	require.Nil(t, err)
	assert.Equal(t, additional, *hosts[0].Additional)

	hosts, err = ds.ListHosts(kolide.HostListOptions{AdditionalFilters: []string{"field1", "field2"}})
	require.Nil(t, err)
	assert.Equal(t, additional, *hosts[0].Additional)

	hosts, err = ds.ListHosts(kolide.HostListOptions{})
	require.Nil(t, err)
	assert.Equal(t, additional, *hosts[0].Additional)

	additional = json.RawMessage(`{"field1": "v1", "missing": null}`)
	hosts, err = ds.ListHosts(kolide.HostListOptions{AdditionalFilters: []string{"field1", "missing"}})
	require.Nil(t, err)
	assert.Equal(t, additional, *hosts[0].Additional)
}

func testListHostsStatus(t *testing.T, ds kolide.Datastore) {
	for i := 0; i < 10; i++ {
		_, err := ds.NewHost(&kolide.Host{
			DetailUpdateTime: time.Now(),
			LabelUpdateTime:  time.Now(),
			SeenTime:         time.Now().Add(-time.Duration(i) * time.Minute),
			OsqueryHostID:    strconv.Itoa(i),
			NodeKey:          fmt.Sprintf("%d", i),
			UUID:             fmt.Sprintf("%d", i),
			HostName:         fmt.Sprintf("foo.local%d", i),
		})
		assert.Nil(t, err)
		if err != nil {
			return
		}
	}

	hosts, err := ds.ListHosts(kolide.HostListOptions{StatusFilter: "online"})
	require.Nil(t, err)
	assert.Equal(t, 1, len(hosts))

	hosts, err = ds.ListHosts(kolide.HostListOptions{StatusFilter: "offline"})
	require.Nil(t, err)
	assert.Equal(t, 9, len(hosts))

	hosts, err = ds.ListHosts(kolide.HostListOptions{StatusFilter: "mia"})
	require.Nil(t, err)
	assert.Equal(t, 0, len(hosts))

	hosts, err = ds.ListHosts(kolide.HostListOptions{StatusFilter: "new"})
	require.Nil(t, err)
	assert.Equal(t, 10, len(hosts))
}

func testListHostsQuery(t *testing.T, ds kolide.Datastore) {
	hosts := []*kolide.Host{}
	for i := 0; i < 10; i++ {
		host, err := ds.NewHost(&kolide.Host{
			DetailUpdateTime: time.Now(),
			LabelUpdateTime:  time.Now(),
			SeenTime:         time.Now(),
			OsqueryHostID:    strconv.Itoa(i),
			NodeKey:          fmt.Sprintf("%d", i),
			UUID:             fmt.Sprintf("uuid_00%d", i),
			HostName:         fmt.Sprintf("hostname%%00%d", i),
			HardwareSerial:   fmt.Sprintf("serial00%d", i),
		})
		require.NoError(t, err)
		host.PrimaryIP = fmt.Sprintf("192.168.1.%d", i)
		require.NoError(t, ds.SaveHost(host))
		hosts = append(hosts, host)
	}

	gotHosts, err := ds.ListHosts(kolide.HostListOptions{})
	require.Nil(t, err)
	assert.Equal(t, len(hosts), len(gotHosts))

	gotHosts, err = ds.ListHosts(kolide.HostListOptions{MatchQuery: "00"})
	require.Nil(t, err)
	assert.Equal(t, 10, len(gotHosts))

	gotHosts, err = ds.ListHosts(kolide.HostListOptions{MatchQuery: "000"})
	require.Nil(t, err)
	assert.Equal(t, 1, len(gotHosts))

	gotHosts, err = ds.ListHosts(kolide.HostListOptions{MatchQuery: "192.168."})
	require.Nil(t, err)
	assert.Equal(t, 10, len(gotHosts))

	gotHosts, err = ds.ListHosts(kolide.HostListOptions{MatchQuery: "192.168.1.1"})
	require.Nil(t, err)
	assert.Equal(t, 1, len(gotHosts))

	gotHosts, err = ds.ListHosts(kolide.HostListOptions{MatchQuery: "hostname%00"})
	require.Nil(t, err)
	assert.Equal(t, 10, len(gotHosts))

	gotHosts, err = ds.ListHosts(kolide.HostListOptions{MatchQuery: "hostname%003"})
	require.Nil(t, err)
	assert.Equal(t, 1, len(gotHosts))

	gotHosts, err = ds.ListHosts(kolide.HostListOptions{MatchQuery: "uuid_"})
	require.Nil(t, err)
	assert.Equal(t, 10, len(gotHosts))

	gotHosts, err = ds.ListHosts(kolide.HostListOptions{MatchQuery: "uuid_006"})
	require.Nil(t, err)
	assert.Equal(t, 1, len(gotHosts))

	gotHosts, err = ds.ListHosts(kolide.HostListOptions{MatchQuery: "serial"})
	require.Nil(t, err)
	assert.Equal(t, 10, len(gotHosts))

	gotHosts, err = ds.ListHosts(kolide.HostListOptions{MatchQuery: "serial009"})
	require.Nil(t, err)
	assert.Equal(t, 1, len(gotHosts))
}

func testEnrollHost(t *testing.T, ds kolide.Datastore) {
	test.AddAllHostsLabel(t, ds)
	enrollSecretName := "default"
	for _, tt := range enrollTests {
		h, err := ds.EnrollHost(tt.uuid, tt.nodeKey, enrollSecretName, 0)
		require.Nil(t, err)

		assert.Equal(t, tt.uuid, h.OsqueryHostID)
		assert.Equal(t, tt.nodeKey, h.NodeKey)
		assert.Equal(t, enrollSecretName, h.EnrollSecretName)

		// This host should be allowed to re-enroll immediately if cooldown is disabled
		_, err = ds.EnrollHost(tt.uuid, tt.nodeKey+"new", enrollSecretName+"new", 0)
		require.NoError(t, err)

		// This host should not be allowed to re-enroll immediately if cooldown is enabled
		_, err = ds.EnrollHost(tt.uuid, tt.nodeKey+"new", enrollSecretName+"new", 10*time.Second)
		require.Error(t, err)
	}
}

func testAuthenticateHost(t *testing.T, ds kolide.Datastore) {
	test.AddAllHostsLabel(t, ds)
	for _, tt := range enrollTests {
		h, err := ds.EnrollHost(tt.uuid, tt.nodeKey, "default", 0)
		require.Nil(t, err)

		returned, err := ds.AuthenticateHost(h.NodeKey)
		require.Nil(t, err)
		assert.Equal(t, h.NodeKey, returned.NodeKey)
	}

	_, err := ds.AuthenticateHost("7B1A9DC9-B042-489F-8D5A-EEC2412C95AA")
	assert.NotNil(t, err)

	_, err = ds.AuthenticateHost("")
	assert.NotNil(t, err)
}

func testAuthenticateHostCaseSensitive(t *testing.T, ds kolide.Datastore) {
	test.AddAllHostsLabel(t, ds)
	for _, tt := range enrollTests {
		h, err := ds.EnrollHost(tt.uuid, tt.nodeKey, "default", 0)
		require.Nil(t, err)

		_, err = ds.AuthenticateHost(strings.ToUpper(h.NodeKey))
		require.Error(t, err, "node key authentication should be case sensitive")
	}
}

func testSearchHosts(t *testing.T, ds kolide.Datastore) {
	_, err := ds.NewHost(&kolide.Host{
		OsqueryHostID:    "1234",
		DetailUpdateTime: time.Now(),
		LabelUpdateTime:  time.Now(),
		SeenTime:         time.Now(),
		NodeKey:          "1",
		UUID:             "1",
		HostName:         "foo.local",
	})
	require.Nil(t, err)

	h2, err := ds.NewHost(&kolide.Host{
		OsqueryHostID:    "5679",
		DetailUpdateTime: time.Now(),
		LabelUpdateTime:  time.Now(),
		SeenTime:         time.Now(),
		NodeKey:          "2",
		UUID:             "2",
		HostName:         "bar.local",
	})
	require.Nil(t, err)

	h3, err := ds.NewHost(&kolide.Host{
		OsqueryHostID:    "99999",
		DetailUpdateTime: time.Now(),
		LabelUpdateTime:  time.Now(),
		SeenTime:         time.Now(),
		NodeKey:          "3",
		UUID:             "abc-def-ghi",
		HostName:         "foo-bar.local",
	})
	require.Nil(t, err)

	// We once threw errors when the search query was empty. Verify that we
	// don't error.
	_, err = ds.SearchHosts("")
	require.Nil(t, err)

	hosts, err := ds.SearchHosts("foo")
	assert.Nil(t, err)
	assert.Len(t, hosts, 2)

	host, err := ds.SearchHosts("foo", h3.ID)
	require.Nil(t, err)
	require.Len(t, host, 1)
	assert.Equal(t, "foo.local", host[0].HostName)

	host, err = ds.SearchHosts("foo", h3.ID, h2.ID)
	require.Nil(t, err)
	require.Len(t, host, 1)
	assert.Equal(t, "foo.local", host[0].HostName)

	host, err = ds.SearchHosts("abc")
	require.Nil(t, err)
	require.Len(t, host, 1)
	assert.Equal(t, "abc-def-ghi", host[0].UUID)

	none, err := ds.SearchHosts("xxx")
	assert.Nil(t, err)
	assert.Len(t, none, 0)

	// check to make sure search on ip address works
	h2.PrimaryIP = "99.100.101.103"
	err = ds.SaveHost(h2)
	require.Nil(t, err)

	hits, err := ds.SearchHosts("99.100.101")
	require.Nil(t, err)
	require.Equal(t, 1, len(hits))

	hits, err = ds.SearchHosts("99.100.111")
	require.Nil(t, err)
	assert.Equal(t, 0, len(hits))

	h3.PrimaryIP = "99.100.101.104"
	err = ds.SaveHost(h3)
	require.Nil(t, err)
	hits, err = ds.SearchHosts("99.100.101")
	require.Nil(t, err)
	assert.Equal(t, 2, len(hits))
	hits, err = ds.SearchHosts("99.100.101", h3.ID)
	require.Nil(t, err)
	assert.Equal(t, 1, len(hits))
}

func testSearchHostsLimit(t *testing.T, ds kolide.Datastore) {
	for i := 0; i < 15; i++ {
		_, err := ds.NewHost(&kolide.Host{
			DetailUpdateTime: time.Now(),
			LabelUpdateTime:  time.Now(),
			SeenTime:         time.Now(),
			OsqueryHostID:    fmt.Sprintf("host%d", i),
			NodeKey:          fmt.Sprintf("%d", i),
			UUID:             fmt.Sprintf("%d", i),
			HostName:         fmt.Sprintf("foo.%d.local", i),
		})
		require.Nil(t, err)
	}

	hosts, err := ds.SearchHosts("foo")
	require.Nil(t, err)
	assert.Len(t, hosts, 10)
}

func testGenerateHostStatusStatistics(t *testing.T, ds kolide.Datastore) {
	if ds.Name() == "inmem" {
		fmt.Println("Busted test skipped for inmem")
		return
	}

	mockClock := clock.NewMockClock()

	online, offline, mia, new, err := ds.GenerateHostStatusStatistics(mockClock.Now())
	assert.Nil(t, err)
	assert.Equal(t, uint(0), online)
	assert.Equal(t, uint(0), offline)
	assert.Equal(t, uint(0), mia)
	assert.Equal(t, uint(0), new)

	// Online
	h, err := ds.NewHost(&kolide.Host{
		ID:               1,
		OsqueryHostID:    "1",
		NodeKey:          "1",
		DetailUpdateTime: mockClock.Now().Add(-30 * time.Second),
		LabelUpdateTime:  mockClock.Now().Add(-30 * time.Second),
		SeenTime:         mockClock.Now().Add(-30 * time.Second),
	})
	require.Nil(t, err)
	h.DistributedInterval = 15
	h.ConfigTLSRefresh = 30
	require.Nil(t, ds.SaveHost(h))

	// Online
	h, err = ds.NewHost(&kolide.Host{
		ID:               2,
		OsqueryHostID:    "2",
		NodeKey:          "2",
		DetailUpdateTime: mockClock.Now().Add(-1 * time.Minute),
		LabelUpdateTime:  mockClock.Now().Add(-1 * time.Minute),
		SeenTime:         mockClock.Now().Add(-1 * time.Minute),
	})
	require.Nil(t, err)
	h.DistributedInterval = 60
	h.ConfigTLSRefresh = 3600
	require.Nil(t, ds.SaveHost(h))

	// Offline
	h, err = ds.NewHost(&kolide.Host{
		ID:               3,
		OsqueryHostID:    "3",
		NodeKey:          "3",
		DetailUpdateTime: mockClock.Now().Add(-1 * time.Hour),
		LabelUpdateTime:  mockClock.Now().Add(-1 * time.Hour),
		SeenTime:         mockClock.Now().Add(-1 * time.Hour),
	})
	require.Nil(t, err)
	h.DistributedInterval = 300
	h.ConfigTLSRefresh = 300
	require.Nil(t, ds.SaveHost(h))

	// MIA
	h, err = ds.NewHost(&kolide.Host{
		ID:               4,
		OsqueryHostID:    "4",
		NodeKey:          "4",
		DetailUpdateTime: mockClock.Now().Add(-35 * (24 * time.Hour)),
		LabelUpdateTime:  mockClock.Now().Add(-35 * (24 * time.Hour)),
		SeenTime:         mockClock.Now().Add(-35 * (24 * time.Hour)),
	})
	require.Nil(t, err)

	online, offline, mia, new, err = ds.GenerateHostStatusStatistics(mockClock.Now())
	assert.Nil(t, err)
	assert.Equal(t, uint(2), online)
	assert.Equal(t, uint(1), offline)
	assert.Equal(t, uint(1), mia)
	assert.Equal(t, uint(4), new)

	online, offline, mia, new, err = ds.GenerateHostStatusStatistics(mockClock.Now().Add(1 * time.Hour))
	assert.Nil(t, err)
	assert.Equal(t, uint(0), online)
	assert.Equal(t, uint(3), offline)
	assert.Equal(t, uint(1), mia)
	assert.Equal(t, uint(4), new)
}

func testMarkHostSeen(t *testing.T, ds kolide.Datastore) {
	mockClock := clock.NewMockClock()

	anHourAgo := mockClock.Now().Add(-1 * time.Hour).UTC()
	aDayAgo := mockClock.Now().Add(-24 * time.Hour).UTC()

	h1, err := ds.NewHost(&kolide.Host{
		ID:               1,
		OsqueryHostID:    "1",
		UUID:             "1",
		NodeKey:          "1",
		DetailUpdateTime: aDayAgo,
		LabelUpdateTime:  aDayAgo,
		SeenTime:         aDayAgo,
	})
	assert.Nil(t, err)

	{
		h1Verify, err := ds.Host(1)
		assert.Nil(t, err)
		require.NotNil(t, h1Verify)
		assert.WithinDuration(t, aDayAgo, h1Verify.SeenTime, time.Second)
	}

	err = ds.MarkHostSeen(h1, anHourAgo)
	assert.Nil(t, err)

	{
		h1Verify, err := ds.Host(1)
		assert.Nil(t, err)
		require.NotNil(t, h1Verify)
		assert.WithinDuration(t, anHourAgo, h1Verify.SeenTime, time.Second)
	}
}

func testMarkHostsSeen(t *testing.T, ds kolide.Datastore) {
	mockClock := clock.NewMockClock()

	aSecondAgo := mockClock.Now().Add(-1 * time.Second).UTC()
	anHourAgo := mockClock.Now().Add(-1 * time.Hour).UTC()
	aDayAgo := mockClock.Now().Add(-24 * time.Hour).UTC()

	h1, err := ds.NewHost(&kolide.Host{
		ID:               1,
		OsqueryHostID:    "1",
		UUID:             "1",
		NodeKey:          "1",
		DetailUpdateTime: aDayAgo,
		LabelUpdateTime:  aDayAgo,
		SeenTime:         aDayAgo,
	})
	require.Nil(t, err)

	h2, err := ds.NewHost(&kolide.Host{
		ID:               2,
		OsqueryHostID:    "2",
		UUID:             "2",
		NodeKey:          "2",
		DetailUpdateTime: aDayAgo,
		LabelUpdateTime:  aDayAgo,
		SeenTime:         aDayAgo,
	})
	require.Nil(t, err)

	err = ds.MarkHostsSeen([]uint{h1.ID}, anHourAgo)
	assert.Nil(t, err)

	{
		h1Verify, err := ds.Host(h1.ID)
		assert.Nil(t, err)
		require.NotNil(t, h1Verify)
		assert.WithinDuration(t, anHourAgo, h1Verify.SeenTime, time.Second)

		h2Verify, err := ds.Host(h2.ID)
		assert.Nil(t, err)
		require.NotNil(t, h2Verify)
		assert.WithinDuration(t, aDayAgo, h2Verify.SeenTime, time.Second)
	}

	err = ds.MarkHostsSeen([]uint{h1.ID, h2.ID}, aSecondAgo)
	assert.Nil(t, err)

	{
		h1Verify, err := ds.Host(h1.ID)
		assert.Nil(t, err)
		require.NotNil(t, h1Verify)
		assert.WithinDuration(t, aSecondAgo, h1Verify.SeenTime, time.Second)

		h2Verify, err := ds.Host(h2.ID)
		assert.Nil(t, err)
		require.NotNil(t, h2Verify)
		assert.WithinDuration(t, aSecondAgo, h2Verify.SeenTime, time.Second)
	}

}

func testCleanupIncomingHosts(t *testing.T, ds kolide.Datastore) {
	mockClock := clock.NewMockClock()

	h1, err := ds.NewHost(&kolide.Host{
		ID:               1,
		OsqueryHostID:    "1",
		UUID:             "1",
		NodeKey:          "1",
		DetailUpdateTime: mockClock.Now(),
		LabelUpdateTime:  mockClock.Now(),
		SeenTime:         mockClock.Now(),
	})
	require.Nil(t, err)

	h2, err := ds.NewHost(&kolide.Host{
		ID:               2,
		OsqueryHostID:    "2",
		UUID:             "2",
		NodeKey:          "2",
		HostName:         "foobar",
		OsqueryVersion:   "3.2.3",
		DetailUpdateTime: mockClock.Now(),
		LabelUpdateTime:  mockClock.Now(),
		SeenTime:         mockClock.Now(),
	})
	require.Nil(t, err)

	err = ds.CleanupIncomingHosts(mockClock.Now().UTC())
	assert.Nil(t, err)

	// Both hosts should still exist because they are new
	_, err = ds.Host(h1.ID)
	assert.Nil(t, err)
	_, err = ds.Host(h2.ID)
	assert.Nil(t, err)

	err = ds.CleanupIncomingHosts(mockClock.Now().Add(6 * time.Minute).UTC())
	assert.Nil(t, err)

	// Now only the host with details should exist
	_, err = ds.Host(h1.ID)
	assert.NotNil(t, err)
	_, err = ds.Host(h2.ID)
	assert.Nil(t, err)
}

func testHostIDsByName(t *testing.T, ds kolide.Datastore) {
	for i := 0; i < 10; i++ {
		_, err := ds.NewHost(&kolide.Host{
			DetailUpdateTime: time.Now(),
			LabelUpdateTime:  time.Now(),
			SeenTime:         time.Now(),
			OsqueryHostID:    fmt.Sprintf("host%d", i),
			NodeKey:          fmt.Sprintf("%d", i),
			UUID:             fmt.Sprintf("%d", i),
			HostName:         fmt.Sprintf("foo.%d.local", i),
		})
		require.Nil(t, err)
	}

	hosts, err := ds.HostIDsByName([]string{"foo.2.local", "foo.1.local", "foo.5.local"})
	require.Nil(t, err)
	sort.Slice(hosts, func(i, j int) bool { return hosts[i] < hosts[j] })
	assert.Equal(t, hosts, []uint{2, 3, 6})
}

func testHostAdditional(t *testing.T, ds kolide.Datastore) {
	_, err := ds.NewHost(&kolide.Host{
		DetailUpdateTime: time.Now(),
		LabelUpdateTime:  time.Now(),
		SeenTime:         time.Now(),
		OsqueryHostID:    "foobar",
		NodeKey:          "nodekey",
		UUID:             "uuid",
		HostName:         "foobar.local",
	})
	require.Nil(t, err)

	h, err := ds.AuthenticateHost("nodekey")
	require.Nil(t, err)
	assert.Equal(t, "foobar.local", h.HostName)
	assert.Nil(t, h.Additional)

	// Additional not yet set
	h, err = ds.Host(h.ID)
	require.Nil(t, err)
	assert.Nil(t, h.Additional)

	// Add additional
	additional := json.RawMessage(`{"additional": "result"}`)
	h.Additional = &additional
	err = ds.SaveHost(h)
	require.Nil(t, err)

	h, err = ds.AuthenticateHost("nodekey")
	require.Nil(t, err)
	assert.Equal(t, "foobar.local", h.HostName)
	assert.Nil(t, h.Additional)

	h, err = ds.Host(h.ID)
	require.Nil(t, err)
	assert.Equal(t, additional, *h.Additional)

	// Update besides additional. Additional should be unchanged.
	h, err = ds.AuthenticateHost("nodekey")
	require.Nil(t, err)
	h.HostName = "baz.local"
	err = ds.SaveHost(h)
	require.Nil(t, err)

	h, err = ds.AuthenticateHost("nodekey")
	require.Nil(t, err)
	assert.Equal(t, "baz.local", h.HostName)
	assert.Nil(t, h.Additional)

	h, err = ds.Host(h.ID)
	require.Nil(t, err)
	assert.Equal(t, additional, *h.Additional)

	// Update additional
	additional = json.RawMessage(`{"other": "additional"}`)
	h, err = ds.AuthenticateHost("nodekey")
	require.Nil(t, err)
	h.Additional = &additional
	err = ds.SaveHost(h)
	require.Nil(t, err)

	h, err = ds.AuthenticateHost("nodekey")
	require.Nil(t, err)
	assert.Equal(t, "baz.local", h.HostName)
	assert.Nil(t, h.Additional)

	h, err = ds.Host(h.ID)
	require.Nil(t, err)
	assert.Equal(t, additional, *h.Additional)
}

func testHostByIdentifier(t *testing.T, ds kolide.Datastore) {
	for i := 1; i <= 10; i++ {
		_, err := ds.NewHost(&kolide.Host{
			DetailUpdateTime: time.Now(),
			LabelUpdateTime:  time.Now(),
			SeenTime:         time.Now(),
			OsqueryHostID:    fmt.Sprintf("osquery_host_id_%d", i),
			NodeKey:          fmt.Sprintf("node_key_%d", i),
			UUID:             fmt.Sprintf("uuid_%d", i),
			HostName:         fmt.Sprintf("hostname_%d", i),
		})
		require.Nil(t, err)
	}

	var (
		h   *kolide.Host
		err error
	)
	h, err = ds.HostByIdentifier("uuid_1")
	require.NoError(t, err)
	assert.Equal(t, uint(1), h.ID)

	h, err = ds.HostByIdentifier("osquery_host_id_2")
	require.NoError(t, err)
	assert.Equal(t, uint(2), h.ID)

	h, err = ds.HostByIdentifier("node_key_4")
	require.NoError(t, err)
	assert.Equal(t, uint(4), h.ID)

	h, err = ds.HostByIdentifier("hostname_7")
	require.NoError(t, err)
	assert.Equal(t, uint(7), h.ID)

	h, err = ds.HostByIdentifier("foobar")
	require.Error(t, err)
}
