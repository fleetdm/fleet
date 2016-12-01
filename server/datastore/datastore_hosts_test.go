package datastore

import (
	"fmt"
	"testing"
	"time"

	"github.com/kolide/kolide-ose/server/kolide"
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

func testSaveHosts(t *testing.T, db kolide.Datastore) {
	host, err := db.NewHost(&kolide.Host{
		DetailUpdateTime: time.Now(),
		NodeKey:          "1",
		UUID:             "1",
		HostName:         "foo.local",
	})
	require.Nil(t, err)
	require.NotNil(t, host)

	host.HostName = "bar.local"
	err = db.SaveHost(host)
	require.Nil(t, err)

	host, err = db.Host(host.ID)
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

	err = db.SaveHost(host)
	require.Nil(t, err)

	host, err = db.Host(host.ID)
	require.Nil(t, err)
	require.NotNil(t, host)
	require.Equal(t, 2, len(host.NetworkInterfaces))
	primaryNicID := host.NetworkInterfaces[0].ID
	host.PrimaryNetworkInterfaceID = &primaryNicID
	err = db.SaveHost(host)
	require.Nil(t, err)
	host, err = db.Host(host.ID)
	require.Nil(t, err)
	require.NotNil(t, host)
	require.Equal(t, 2, len(host.NetworkInterfaces))
	assert.Equal(t, primaryNicID, *host.PrimaryNetworkInterfaceID)

	// remove primary nic, host primary nic should change
	host.NetworkInterfaces = []*kolide.NetworkInterface{
		host.NetworkInterfaces[1],
	}
	err = db.SaveHost(host)
	require.Nil(t, err)
	host, err = db.Host(host.ID)
	require.Nil(t, err)
	require.NotNil(t, host)
	assert.Equal(t, host.NetworkInterfaces[0].ID, *host.PrimaryNetworkInterfaceID)
	assert.Equal(t, 1, len(host.NetworkInterfaces))

	err = db.DeleteHost(host)
	assert.Nil(t, err)

	host, err = db.Host(host.ID)
	assert.NotNil(t, err)
	assert.Nil(t, host)
}

func testDeleteHost(t *testing.T, db kolide.Datastore) {
	host, err := db.NewHost(&kolide.Host{
		DetailUpdateTime: time.Now(),
		NodeKey:          "1",
		UUID:             "1",
		HostName:         "foo.local",
	})
	assert.Nil(t, err)
	assert.NotNil(t, host)

	err = db.DeleteHost(host)
	assert.Nil(t, err)

	host, err = db.Host(host.ID)
	assert.NotNil(t, err)
}

func testListHost(t *testing.T, db kolide.Datastore) {
	hosts := []*kolide.Host{}
	for i := 0; i < 10; i++ {
		host, err := db.NewHost(&kolide.Host{
			DetailUpdateTime: time.Now(),
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

	hosts2, err := db.ListHosts(kolide.ListOptions{})
	require.Nil(t, err)
	assert.Equal(t, len(hosts), len(hosts2))
	err = db.DeleteHost(hosts[0])
	require.Nil(t, err)
	hosts2, err = db.ListHosts(kolide.ListOptions{})
	require.Nil(t, err)
	assert.Equal(t, len(hosts)-1, len(hosts2))

	hosts, err = db.ListHosts(kolide.ListOptions{})
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

	err = db.SaveHost(hosts[0])
	require.Nil(t, err)
	hosts2, err = db.ListHosts(kolide.ListOptions{})
	require.Nil(t, err)
	require.Equal(t, hosts[0].ID, hosts2[0].ID)
	assert.Equal(t, len(hosts[0].NetworkInterfaces), len(hosts2[0].NetworkInterfaces))
	assert.Equal(t, 0, len(hosts2[1].NetworkInterfaces))
	assert.Equal(t, hosts[0].ID, hosts2[0].NetworkInterfaces[0].HostID)
}

func testEnrollHost(t *testing.T, db kolide.Datastore) {
	var hosts []*kolide.Host
	for _, tt := range enrollTests {
		h, err := db.EnrollHost(tt.uuid, tt.hostname, tt.platform, tt.nodeKeySize)
		require.Nil(t, err)

		hosts = append(hosts, h)
		assert.Equal(t, tt.uuid, h.UUID)
		assert.Equal(t, tt.hostname, h.HostName)
		assert.Equal(t, tt.platform, h.Platform)
		assert.NotEmpty(t, h.NodeKey)
	}

	for _, enrolled := range hosts {
		oldNodeKey := enrolled.NodeKey
		newhostname := fmt.Sprintf("changed.%s", enrolled.HostName)

		h, err := db.EnrollHost(enrolled.UUID, newhostname, enrolled.Platform, 15)
		assert.Nil(t, err)
		assert.Equal(t, enrolled.UUID, h.UUID)
		assert.NotEmpty(t, h.NodeKey)
		assert.NotEqual(t, oldNodeKey, h.NodeKey)
	}

}

func testAuthenticateHost(t *testing.T, db kolide.Datastore) {
	for _, tt := range enrollTests {
		h, err := db.EnrollHost(tt.uuid, tt.hostname, tt.platform, tt.nodeKeySize)
		require.Nil(t, err)

		returned, err := db.AuthenticateHost(h.NodeKey)
		require.Nil(t, err)
		assert.Equal(t, h.NodeKey, returned.NodeKey)
	}

	_, err := db.AuthenticateHost("7B1A9DC9-B042-489F-8D5A-EEC2412C95AA")
	assert.NotNil(t, err)

	_, err = db.AuthenticateHost("")
	assert.NotNil(t, err)
}

func testSearchHosts(t *testing.T, db kolide.Datastore) {
	_, err := db.NewHost(&kolide.Host{
		DetailUpdateTime: time.Now(),
		NodeKey:          "1",
		UUID:             "1",
		HostName:         "foo.local",
	})
	require.Nil(t, err)

	h2, err := db.NewHost(&kolide.Host{
		DetailUpdateTime: time.Now(),
		NodeKey:          "2",
		UUID:             "2",
		HostName:         "bar.local",
	})
	require.Nil(t, err)

	h3, err := db.NewHost(&kolide.Host{
		DetailUpdateTime: time.Now(),
		NodeKey:          "3",
		UUID:             "3",
		HostName:         "foo-bar.local",
	})
	require.Nil(t, err)

	hosts, err := db.SearchHosts("foo")
	assert.Nil(t, err)
	assert.Len(t, hosts, 2)

	host, err := db.SearchHosts("foo", h3.ID)
	require.Nil(t, err)
	require.Len(t, host, 1)
	assert.Equal(t, "foo.local", host[0].HostName)

	host, err = db.SearchHosts("foo", h3.ID, h2.ID)
	require.Nil(t, err)
	require.Len(t, host, 1)
	assert.Equal(t, "foo.local", host[0].HostName)

	none, err := db.SearchHosts("xxx")
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
	err = db.SaveHost(h2)
	require.Nil(t, err)

	hits, err := db.SearchHosts("99.100.101")
	require.Nil(t, err)
	require.Equal(t, 1, len(hits))
	assert.Equal(t, 2, len(hits[0].NetworkInterfaces))

	hits, err = db.SearchHosts("99.100.111")
	require.Nil(t, err)
	assert.Equal(t, 0, len(hits))

	h3.NetworkInterfaces = []*kolide.NetworkInterface{
		&kolide.NetworkInterface{
			Interface: "en0",
			IPAddress: "99.100.101.104",
		},
	}
	err = db.SaveHost(h3)
	require.Nil(t, err)
	hits, err = db.SearchHosts("99.100.101")
	require.Nil(t, err)
	assert.Equal(t, 2, len(hits))

	hits, err = db.SearchHosts("99.100.101", h3.ID)
	require.Nil(t, err)
	assert.Equal(t, 1, len(hits))

}

func testSearchHostsLimit(t *testing.T, db kolide.Datastore) {
	for i := 0; i < 15; i++ {
		_, err := db.NewHost(&kolide.Host{
			DetailUpdateTime: time.Now(),
			NodeKey:          fmt.Sprintf("%d", i),
			UUID:             fmt.Sprintf("%d", i),
			HostName:         fmt.Sprintf("foo.%d.local", i),
		})
		require.Nil(t, err)
	}

	hosts, err := db.SearchHosts("foo")
	require.Nil(t, err)
	assert.Len(t, hosts, 10)
}

func testDistributedQueriesForHost(t *testing.T, db kolide.Datastore) {
	h1, err := db.NewHost(&kolide.Host{
		DetailUpdateTime: time.Now(),
		NodeKey:          "1",
		UUID:             "1",
		HostName:         "foo.local",
	})
	require.Nil(t, err)

	h2, err := db.NewHost(&kolide.Host{
		DetailUpdateTime: time.Now(),
		NodeKey:          "2",
		UUID:             "2",
		HostName:         "bar.local",
	})
	require.Nil(t, err)

	// All should have no queries
	var queries map[uint]string
	queries, err = db.DistributedQueriesForHost(h1)
	require.Nil(t, err)
	assert.Empty(t, queries)
	queries, err = db.DistributedQueriesForHost(h2)
	require.Nil(t, err)
	assert.Empty(t, queries)

	// Create a label
	l1, err := db.NewLabel(&kolide.Label{
		Name:  "label foo",
		Query: "query1",
	})
	require.Nil(t, err)
	l1ID := fmt.Sprintf("%d", l1.ID)

	// Add hosts to label
	for _, h := range []*kolide.Host{h1, h2} {
		err = db.RecordLabelQueryExecutions(h, map[string]bool{l1ID: true}, time.Now())
		require.Nil(t, err)
	}

	// Create a query
	q1 := &kolide.Query{
		Name:  "bar",
		Query: "select * from bar",
	}
	q1, err = db.NewQuery(q1)
	require.Nil(t, err)

	// Create a query campaign
	c1 := &kolide.DistributedQueryCampaign{
		QueryID: q1.ID,
		Status:  kolide.QueryRunning,
	}
	c1, err = db.NewDistributedQueryCampaign(c1)
	require.Nil(t, err)

	// Add a target to the campaign
	target := &kolide.DistributedQueryCampaignTarget{
		Type: kolide.TargetLabel,
		DistributedQueryCampaignID: c1.ID,
		TargetID:                   l1.ID,
	}
	target, err = db.NewDistributedQueryCampaignTarget(target)
	require.Nil(t, err)

	// All should have the query now
	queries, err = db.DistributedQueriesForHost(h1)
	require.Nil(t, err)
	assert.Len(t, queries, 1)
	assert.Equal(t, "select * from bar", queries[c1.ID])
	queries, err = db.DistributedQueriesForHost(h2)
	require.Nil(t, err)
	assert.Len(t, queries, 1)
	assert.Equal(t, "select * from bar", queries[c1.ID])

	// Record an execution
	exec := &kolide.DistributedQueryExecution{
		HostID: h1.ID,
		DistributedQueryCampaignID: c1.ID,
		Status: kolide.ExecutionSucceeded,
	}
	_, err = db.NewDistributedQueryExecution(exec)
	require.Nil(t, err)

	// Add another query/campaign
	q2 := &kolide.Query{
		Name:  "foo",
		Query: "select * from foo",
	}
	q2, err = db.NewQuery(q2)
	require.Nil(t, err)

	c2 := &kolide.DistributedQueryCampaign{
		QueryID: q2.ID,
		Status:  kolide.QueryRunning,
	}
	c2, err = db.NewDistributedQueryCampaign(c2)
	require.Nil(t, err)

	// This one targets only h1
	target = &kolide.DistributedQueryCampaignTarget{
		Type: kolide.TargetHost,
		DistributedQueryCampaignID: c2.ID,
		TargetID:                   h1.ID,
	}
	_, err = db.NewDistributedQueryCampaignTarget(target)
	require.Nil(t, err)

	// Check for correct queries
	queries, err = db.DistributedQueriesForHost(h1)
	require.Nil(t, err)
	assert.Len(t, queries, 1)
	assert.Equal(t, "select * from foo", queries[c2.ID])
	queries, err = db.DistributedQueriesForHost(h2)
	require.Nil(t, err)
	assert.Len(t, queries, 1)
	assert.Equal(t, "select * from bar", queries[c1.ID])

	// End both of the campaigns
	c1.Status = kolide.QueryComplete
	require.Nil(t, db.SaveDistributedQueryCampaign(c1))
	c2.Status = kolide.QueryComplete
	require.Nil(t, db.SaveDistributedQueryCampaign(c2))

	// Now no queries should be returned
	queries, err = db.DistributedQueriesForHost(h1)
	require.Nil(t, err)
	assert.Empty(t, queries)
	queries, err = db.DistributedQueriesForHost(h2)
	require.Nil(t, err)
	assert.Empty(t, queries)

}
