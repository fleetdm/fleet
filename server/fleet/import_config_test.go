package fleet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigUnmarshalling(t *testing.T) {
	contents := `
	{
	"options":null,
	"schedule":null,
	"packs":{
		"internal_stuff":{
			"discovery":["select pid from processes where name = 'ldap';"],
			"platform":"linux",
			"queries":{
				"active_directory":{
					"description":"Check each user's active directory cached settings.",
					"interval":"1200",
					"query":"select * from ad_config;"
					}
				},
				"version":"1.5.2"
			},
			"testing":{
				"queries":{
					"suid_bins":{
						"interval":"3600",
						"query":"select * from suid_bins;"
					}
				},
				"shard":"10"
			}
		},
		"file_paths":null,
		"yara":null,
		"prometheus_targets":null,
		"decorators":null
	}
	`

	conf := ImportConfig{
		Packs:         make(PackNameMap),
		ExternalPacks: make(PackNameToPackDetails),
	}

	err := json.Unmarshal([]byte(contents), &conf)
	assert.Nil(t, err)
	require.NotNil(t, conf.Packs["testing"])
	// platform is not defined in the testing pack, so, per osquery docs
	// we default to 'all' platforms
	details, ok := conf.Packs["testing"].(PackDetails)
	require.True(t, ok)
	assert.Equal(t, "", details.Platform)
}

func TestIntervalUnmarshal(t *testing.T) {
	scenarios := []struct {
		name           string
		testVal        interface{}
		errExpected    bool
		expectedResult OsQueryConfigInt
	}{
		{"string to uint", "100", false, 100},
		{"float to uint", float64(123), false, 123},
		{"nil to zero value int", nil, false, 0},
		{"invalid string", "hi there", true, 0},
	}
	for _, scenario := range scenarios {
		t.Run(fmt.Sprintf(": %s", scenario.name), func(tt *testing.T) {
			v, e := unmarshalInteger(scenario.testVal)
			if scenario.errExpected {
				assert.NotNil(t, e)
			} else {
				require.Nil(t, e)
				assert.Equal(t, scenario.expectedResult, v)
			}
		})
	}
}

type importIntTest struct {
	Val OsQueryConfigInt `json:"val"`
}

func TestConfigImportInt(t *testing.T) {
	buff := bytes.NewBufferString(`{"val":"23"}`)
	var ts importIntTest
	err := json.NewDecoder(buff).Decode(&ts)
	assert.Nil(t, err)
	assert.Equal(t, 23, int(ts.Val))

	buff = bytes.NewBufferString(`{"val":456}`)
	err = json.NewDecoder(buff).Decode(&ts)
	assert.Nil(t, err)
	assert.Equal(t, 456, int(ts.Val))

	buff = bytes.NewBufferString(`{"val":"hi 456"}`)
	err = json.NewDecoder(buff).Decode(&ts)
	assert.NotNil(t, err)

}

func TestPackNameMapUnmarshal(t *testing.T) {
	s2p := func(s string) *string { return &s }
	u2p := func(ui uint) *OsQueryConfigInt { ci := OsQueryConfigInt(ui); return &ci }

	pnm := PackNameMap{
		"path": "/this/is/a/path",
		"details": PackDetails{
			Queries: QueryNameToQueryDetailsMap{
				"q1": QueryDetails{
					Query:    "select from foo",
					Interval: 100,
					Removed:  new(bool),
					Platform: s2p("linux"),
					Shard:    new(OsQueryConfigInt),
					Snapshot: new(bool),
				},
			},
			Discovery: []string{
				"select from something",
			},
		},
	}
	b, _ := json.Marshal(pnm)
	actual := make(PackNameMap)
	err := json.Unmarshal(b, &actual)
	require.Nil(t, err)
	assert.Len(t, actual, 2)

	pnm = PackNameMap{
		"path": "/this/is/a/path",
		"details": PackDetails{
			Queries: QueryNameToQueryDetailsMap{
				"q1": QueryDetails{
					Query:    "select from foo",
					Interval: 100,
					Removed:  new(bool),
					Platform: s2p("linux"),
					Shard:    new(OsQueryConfigInt),
					Snapshot: new(bool),
				},
			},
			Shard:    u2p(10),
			Version:  s2p("1.0"),
			Platform: "linux",
			Discovery: []string{
				"select from something",
			},
		},
		"details2": PackDetails{
			Queries: QueryNameToQueryDetailsMap{
				"q1": QueryDetails{
					Query:    "select from bar",
					Interval: 100,
					Removed:  new(bool),
					Platform: s2p("linux"),
					Shard:    new(OsQueryConfigInt),
					Snapshot: new(bool),
				},
			},
			Shard:    u2p(10),
			Version:  s2p("1.0"),
			Platform: "linux",
		},
	}

	b, _ = json.Marshal(pnm)
	actual = make(PackNameMap)
	err = json.Unmarshal(b, &actual)
	require.Nil(t, err)
	assert.Len(t, actual, 3)
}
