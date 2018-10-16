package datastore

import (
	"fmt"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/kolide/fleet/server/kolide"
	"github.com/kolide/fleet/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var enrollTests = []struct {
	uuid, hostname, platform string
	nodeKeySize              int
}{
	0: {uuid: "6D14C88F-8ECF-48D5-9197-777647BF6B26",
		hostname:    "web.kolide.co",
		platform:    "linux",
		nodeKeySize: 12,
	},
	1: {uuid: "B998C0EB-38CE-43B1-A743-FBD7A5C9513B",
		hostname:    "mail.kolide.co",
		platform:    "linux",
		nodeKeySize: 10,
	},
	2: {uuid: "008F0688-5311-4C59-86EE-00C2D6FC3EC2",
		hostname:    "home.kolide.co",
		platform:    "darwin",
		nodeKeySize: 25,
	},
	3: {uuid: "uuid123",
		hostname:    "fakehostname",
		platform:    "darwin",
		nodeKeySize: 1,
	},
}

func testSaveHosts(t *testing.T, ds kolide.Datastore) {
	host, err := ds.NewHost(&kolide.Host{
		DetailUpdateTime: time.Now(),
		SeenTime:         time.Now(),
		NodeKey:          "1",
		UUID:             "1",
		HostName:         "foo.local",
	})
	require.Nil(t, err)
	require.NotNil(t, host)

	host.HostName = "bar.local"
	err = ds.SaveHost(host)
	require.Nil(t, err)

	host, err = ds.Host(host.ID)
	require.Nil(t, err)
	assert.Equal(t, "bar.local", host.HostName)

	host.NetworkInterfaces = []*kolide.NetworkInterface{
		&kolide.NetworkInterface{
			HostID:    host.ID,
			Interface: "en0",
			IPAddress: "98.99.100.101",
		},
		&kolide.NetworkInterface{
			HostID:    host.ID,
			Interface: "en1",
			IPAddress: "98.99.100.102",
		},
	}

	err = ds.SaveHost(host)
	require.Nil(t, err)

	host, err = ds.Host(host.ID)
	require.Nil(t, err)
	require.NotNil(t, host)
	require.Equal(t, 2, len(host.NetworkInterfaces))
	primaryNicID := host.NetworkInterfaces[0].ID
	host.PrimaryNetworkInterfaceID = &primaryNicID
	err = ds.SaveHost(host)
	require.Nil(t, err)
	host, err = ds.Host(host.ID)
	require.Nil(t, err)
	require.NotNil(t, host)
	require.Equal(t, 2, len(host.NetworkInterfaces))
	assert.Equal(t, primaryNicID, *host.PrimaryNetworkInterfaceID)

	// remove primary nic, host primary nic should change
	host.NetworkInterfaces = []*kolide.NetworkInterface{
		host.NetworkInterfaces[1],
	}
	err = ds.SaveHost(host)
	require.Nil(t, err)
	host, err = ds.Host(host.ID)
	require.Nil(t, err)
	require.NotNil(t, host)
	assert.Equal(t, host.NetworkInterfaces[0].ID, *host.PrimaryNetworkInterfaceID)
	assert.Equal(t, 1, len(host.NetworkInterfaces))

	// remove all nics primary nic should be nil
	host.NetworkInterfaces = []*kolide.NetworkInterface{}
	err = ds.SaveHost(host)
	require.Nil(t, err)
	assert.Nil(t, host.PrimaryNetworkInterfaceID)
	host, err = ds.Host(host.ID)
	require.Nil(t, err)
	require.NotNil(t, host)
	assert.Nil(t, host.PrimaryNetworkInterfaceID)

	err = ds.DeleteHost(host.ID)
	assert.Nil(t, err)

	host, err = ds.Host(host.ID)
	assert.NotNil(t, err)
	assert.Nil(t, host)
}

func testDeleteHost(t *testing.T, ds kolide.Datastore) {
	host, err := ds.NewHost(&kolide.Host{
		DetailUpdateTime: time.Now(),
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

func testIdempotentDeleteHost(t *testing.T, ds kolide.Datastore) {
	host, err := ds.NewHost(&kolide.Host{
		DetailUpdateTime: time.Now(),
		SeenTime:         time.Now(),
		NodeKey:          "1",
		UUID:             "1",
		HostName:         "foo.local",
	})
	require.Nil(t, err)
	require.NotNil(t, host)
	id := host.ID
	err = ds.DeleteHost(host.ID)

	host, err = ds.Host(host.ID)
	assert.NotNil(t, err)

	err = ds.DeleteHost(id)
	assert.Nil(t, err)
}

func testListHost(t *testing.T, ds kolide.Datastore) {
	hosts := []*kolide.Host{}
	for i := 0; i < 10; i++ {
		host, err := ds.NewHost(&kolide.Host{
			DetailUpdateTime: time.Now(),
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

	hosts[1].NetworkInterfaces = []*kolide.NetworkInterface{
		&kolide.NetworkInterface{
			Interface: "en0",
			IPAddress: "99.100.101.102",
		},
		&kolide.NetworkInterface{
			Interface: "en1",
			IPAddress: "99.100.101.103",
		},
	}

	err := ds.SaveHost(hosts[1])
	require.Nil(t, err)

	hosts[3].NetworkInterfaces = []*kolide.NetworkInterface{
		&kolide.NetworkInterface{
			Interface: "en2",
			IPAddress: "99.100.101.104",
		},
	}
	err = ds.SaveHost(hosts[3])
	require.Nil(t, err)

	hosts2, err := ds.ListHosts(kolide.ListOptions{})
	require.Nil(t, err)
	assert.Equal(t, len(hosts), len(hosts2))

	require.Equal(t, 2, len(hosts2[1].NetworkInterfaces))
	require.Equal(t, 0, len(hosts2[2].NetworkInterfaces))
	require.Equal(t, 1, len(hosts2[3].NetworkInterfaces))
	assert.Equal(t, "en1", hosts2[1].NetworkInterfaces[1].Interface)
	assert.Equal(t, "en2", hosts2[3].NetworkInterfaces[0].Interface)

	// Test with logic for only a few hosts
	hosts2, err = ds.ListHosts(kolide.ListOptions{PerPage: 4, Page: 0})
	require.Nil(t, err)
	assert.Equal(t, 4, len(hosts2))

	require.Equal(t, 2, len(hosts2[1].NetworkInterfaces))
	require.Equal(t, 0, len(hosts2[2].NetworkInterfaces))
	require.Equal(t, 1, len(hosts2[3].NetworkInterfaces))
	assert.Equal(t, "en1", hosts2[1].NetworkInterfaces[1].Interface)
	assert.Equal(t, "en2", hosts2[3].NetworkInterfaces[0].Interface)

	err = ds.DeleteHost(hosts[0].ID)
	require.Nil(t, err)
	hosts2, err = ds.ListHosts(kolide.ListOptions{})
	require.Nil(t, err)
	assert.Equal(t, len(hosts)-1, len(hosts2))

	hosts, err = ds.ListHosts(kolide.ListOptions{})
	require.Nil(t, err)
	require.Equal(t, len(hosts2), len(hosts))
	hosts[0].NetworkInterfaces = []*kolide.NetworkInterface{
		&kolide.NetworkInterface{
			IPAddress: "98.99.100.101",
			Interface: "en0",
		},
		&kolide.NetworkInterface{
			IPAddress: "98.99.100.102",
			Interface: "en1",
		},
	}

	err = ds.SaveHost(hosts[0])
	require.Nil(t, err)
	hosts2, err = ds.ListHosts(kolide.ListOptions{})
	require.Nil(t, err)
	require.Equal(t, hosts[0].ID, hosts2[0].ID)
	assert.Equal(t, len(hosts[0].NetworkInterfaces), len(hosts2[0].NetworkInterfaces))
	assert.Equal(t, 0, len(hosts2[1].NetworkInterfaces))
	assert.Equal(t, hosts[0].ID, hosts2[0].NetworkInterfaces[0].HostID)
}

func testEnrollHost(t *testing.T, ds kolide.Datastore) {
	var hosts []*kolide.Host
	for _, tt := range enrollTests {
		h, err := ds.EnrollHost(tt.uuid, tt.nodeKeySize)
		require.Nil(t, err)

		hosts = append(hosts, h)
		assert.Equal(t, tt.uuid, h.OsqueryHostID)
		assert.NotEmpty(t, h.NodeKey)
	}

}

func testAuthenticateHost(t *testing.T, ds kolide.Datastore) {
	for _, tt := range enrollTests {
		h, err := ds.EnrollHost(tt.uuid, tt.nodeKeySize)
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

func testSearchHosts(t *testing.T, ds kolide.Datastore) {
	_, err := ds.NewHost(&kolide.Host{
		OsqueryHostID:    "1234",
		DetailUpdateTime: time.Now(),
		SeenTime:         time.Now(),
		NodeKey:          "1",
		UUID:             "1",
		HostName:         "foo.local",
	})
	require.Nil(t, err)

	h2, err := ds.NewHost(&kolide.Host{
		OsqueryHostID:    "5679",
		DetailUpdateTime: time.Now(),
		SeenTime:         time.Now(),
		NodeKey:          "2",
		UUID:             "2",
		HostName:         "bar.local",
	})
	require.Nil(t, err)

	h3, err := ds.NewHost(&kolide.Host{
		OsqueryHostID:    "99999",
		DetailUpdateTime: time.Now(),
		SeenTime:         time.Now(),
		NodeKey:          "3",
		UUID:             "3",
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

	none, err := ds.SearchHosts("xxx")
	assert.Nil(t, err)
	assert.Len(t, none, 0)

	// check to make sure search on ip address works
	h2.NetworkInterfaces = []*kolide.NetworkInterface{
		&kolide.NetworkInterface{
			Interface: "en0",
			IPAddress: "99.100.101.102",
		},
		&kolide.NetworkInterface{
			Interface: "en1",
			IPAddress: "99.100.101.103",
		},
	}
	err = ds.SaveHost(h2)
	require.Nil(t, err)

	hits, err := ds.SearchHosts("99.100.101")
	require.Nil(t, err)
	require.Equal(t, 1, len(hits))
	assert.Equal(t, 2, len(hits[0].NetworkInterfaces))

	hits, err = ds.SearchHosts("99.100.111")
	require.Nil(t, err)
	assert.Equal(t, 0, len(hits))

	h3.NetworkInterfaces = []*kolide.NetworkInterface{
		&kolide.NetworkInterface{
			Interface: "en3",
			IPAddress: "99.100.101.104",
		},
	}
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

func testDistributedQueriesForHost(t *testing.T, ds kolide.Datastore) {
	user := test.NewUser(t, ds, "Zach", "zwass", "zwass@kolide.co", true)

	h1, err := ds.NewHost(&kolide.Host{
		OsqueryHostID:    "1",
		DetailUpdateTime: time.Now(),
		SeenTime:         time.Now(),
		NodeKey:          "1",
		UUID:             "1",
		HostName:         "foo.local",
	})
	require.Nil(t, err)

	h2, err := ds.NewHost(&kolide.Host{
		OsqueryHostID:    "2",
		DetailUpdateTime: time.Now(),
		SeenTime:         time.Now(),
		NodeKey:          "2",
		UUID:             "2",
		HostName:         "bar.local",
	})
	require.Nil(t, err)

	// All should have no queries
	var queries map[uint]string
	queries, err = ds.DistributedQueriesForHost(h1)
	require.Nil(t, err)
	assert.Empty(t, queries)
	queries, err = ds.DistributedQueriesForHost(h2)
	require.Nil(t, err)
	assert.Empty(t, queries)

	// Create a label
	l1 := kolide.LabelSpec{
		ID:    1,
		Name:  "label foo",
		Query: "query1",
	}
	err = ds.ApplyLabelSpecs([]*kolide.LabelSpec{&l1})
	require.Nil(t, err)

	// Add hosts to label
	for _, h := range []*kolide.Host{h1, h2} {
		err = ds.RecordLabelQueryExecutions(h, map[uint]bool{l1.ID: true}, time.Now())
		require.Nil(t, err)
	}

	// Create a query
	q1 := &kolide.Query{
		Name:     "bar",
		Query:    "select * from bar",
		AuthorID: &user.ID,
	}
	q1, err = ds.NewQuery(q1)
	require.Nil(t, err)

	// Create a query campaign
	c1 := &kolide.DistributedQueryCampaign{
		QueryID: q1.ID,
		Status:  kolide.QueryRunning,
	}
	c1, err = ds.NewDistributedQueryCampaign(c1)
	require.Nil(t, err)

	// Add a target to the campaign
	target := &kolide.DistributedQueryCampaignTarget{
		Type: kolide.TargetLabel,
		DistributedQueryCampaignID: c1.ID,
		TargetID:                   l1.ID,
	}
	target, err = ds.NewDistributedQueryCampaignTarget(target)
	require.Nil(t, err)

	// All should have the query now
	queries, err = ds.DistributedQueriesForHost(h1)
	require.Nil(t, err)
	assert.Len(t, queries, 1)
	assert.Equal(t, "select * from bar", queries[c1.ID])
	queries, err = ds.DistributedQueriesForHost(h2)
	require.Nil(t, err)
	assert.Len(t, queries, 1)
	assert.Equal(t, "select * from bar", queries[c1.ID])

	// Record an execution
	exec := &kolide.DistributedQueryExecution{
		HostID: h1.ID,
		DistributedQueryCampaignID: c1.ID,
		Status: kolide.ExecutionSucceeded,
	}
	_, err = ds.NewDistributedQueryExecution(exec)
	require.Nil(t, err)

	// Add another query/campaign
	q2 := &kolide.Query{
		Name:     "foo",
		Query:    "select * from foo",
		AuthorID: &user.ID,
	}
	q2, err = ds.NewQuery(q2)
	require.Nil(t, err)

	c2 := &kolide.DistributedQueryCampaign{
		QueryID: q2.ID,
		Status:  kolide.QueryRunning,
	}
	c2, err = ds.NewDistributedQueryCampaign(c2)
	require.Nil(t, err)

	// This one targets only h1
	target = &kolide.DistributedQueryCampaignTarget{
		Type: kolide.TargetHost,
		DistributedQueryCampaignID: c2.ID,
		TargetID:                   h1.ID,
	}
	_, err = ds.NewDistributedQueryCampaignTarget(target)
	require.Nil(t, err)

	// Check for correct queries
	queries, err = ds.DistributedQueriesForHost(h1)
	require.Nil(t, err)
	assert.Len(t, queries, 1)
	assert.Equal(t, "select * from foo", queries[c2.ID])
	queries, err = ds.DistributedQueriesForHost(h2)
	require.Nil(t, err)
	assert.Len(t, queries, 1)
	assert.Equal(t, "select * from bar", queries[c1.ID])

	// End both of the campaigns
	c1.Status = kolide.QueryComplete
	require.Nil(t, ds.SaveDistributedQueryCampaign(c1))
	c2.Status = kolide.QueryComplete
	require.Nil(t, ds.SaveDistributedQueryCampaign(c2))

	// Now no queries should be returned
	queries, err = ds.DistributedQueriesForHost(h1)
	require.Nil(t, err)
	assert.Empty(t, queries)
	queries, err = ds.DistributedQueriesForHost(h2)
	require.Nil(t, err)
	assert.Empty(t, queries)
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

func testFlappingNetworkInterfaces(t *testing.T, ds kolide.Datastore) {
	// See https://github.com/kolide/fleet/issues/1278
	host, err := ds.NewHost(&kolide.Host{
		DetailUpdateTime: time.Now(),
		SeenTime:         time.Now(),
		NodeKey:          "1",
		UUID:             "1",
		HostName:         "foo.local",
	})
	require.Nil(t, err)
	require.NotNil(t, host)

	host.HostName = "bar.local"
	err = ds.SaveHost(host)
	require.Nil(t, err)

	host, err = ds.Host(host.ID)
	require.Nil(t, err)
	assert.Equal(t, "bar.local", host.HostName)

	host.NetworkInterfaces = []*kolide.NetworkInterface{
		&kolide.NetworkInterface{
			HostID:    host.ID,
			Interface: "en0",
			IPAddress: "98.99.100.101",
		},
	}

	err = ds.SaveHost(host)
	require.Nil(t, err)

	host, err = ds.AuthenticateHost(host.NodeKey)
	require.Nil(t, err)
	assert.Len(t, host.NetworkInterfaces, 1)

	// Simulate osquery returning the same results for the network
	// interfaces (note it's important that we reset this so that the ID is
	// 0 before saving)
	host.NetworkInterfaces = []*kolide.NetworkInterface{
		&kolide.NetworkInterface{
			HostID:    host.ID,
			Interface: "en0",
			IPAddress: "98.99.100.101",
		},
	}

	err = ds.SaveHost(host)
	require.Nil(t, err)

	host, err = ds.AuthenticateHost(host.NodeKey)
	require.Nil(t, err)
	assert.Len(t, host.NetworkInterfaces, 1)
}

func testHostIDsByName(t *testing.T, ds kolide.Datastore) {
	for i := 0; i < 10; i++ {
		_, err := ds.NewHost(&kolide.Host{
			DetailUpdateTime: time.Now(),
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
