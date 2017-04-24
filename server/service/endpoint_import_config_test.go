package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/kolide/kolide/server/kolide"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testImportConfigWithGlob(t *testing.T, r *testResource) {
	testJSON := `
{
  "config": "{\"options\":{\"host_identifier\":\"hostname\",\"schedule_splay_percent\":10},\"schedule\":{\"macosx_kextstat\":{\"query\":\"SELECT * FROM kernel_extensions;\",\"interval\":10},\"foobar\":{\"query\":\"SELECT foo, bar, pid FROM foobar_table;\",\"interval\":600}},\"packs\":{\"*\":\"/path/to/glob/*\",\"external_pack\":\"/path/to/external_pack.conf\",\"internal_pack\":{\"discovery\":[\"select pid from processes where name = 'foobar';\",\"select count(*) from users where username like 'www%';\"],\"platform\":\"linux\",\"version\":\"1.5.2\",\"queries\":{\"active_directory\":{\"query\":\"select * from ad_config;\",\"interval\":1200,\"description\":\"Check each user's active directory cached settings.\"}}}},\"decorators\":{\"load\":[\"SELECT version FROM osquery_info\",\"SELECT uuid AS host_uuid FROM system_info\"],\"always\":[\"SELECT user AS username FROM logged_in_users WHERE user <> '' ORDER BY time LIMIT 1;\"],\"interval\":{\"3600\":[\"SELECT total_seconds AS uptime FROM uptime;\"]}},\"glob\":[\"globpack\"],\"yara\":{\"signatures\":{\"sig_group_1\":[\"/Users/wxs/sigs/foo.sig\",\"/Users/wxs/sigs/bar.sig\"],\"sig_group_2\":[\"/Users/wxs/sigs/baz.sig\"]},\"file_paths\":{\"system_binaries\":[\"sig_group_1\"],\"tmp\":[\"sig_group_1\",\"sig_group_2\"]}},\"file_paths\":{\"system_binaries\":[\"/usr/bin/%\",\"/usr/sbin/%\"],\"tmp\":[\"/Users/%/tmp/%%\",\"/tmp/%\"]}}",
  "external_pack_configs": {
    "external_pack": "{\"discovery\":[\"select pid from processes where name = 'baz';\"],\"platform\":\"linux\",\"version\":\"1.5.2\",\"queries\":{\"something\":{\"query\":\"select * from something;\",\"interval\":1200,\"description\":\"Check something.\"}}}",
    "globpack": "{\"discovery\":[\"select pid from processes where name = 'zip';\"],\"platform\":\"linux\",\"version\":\"1.5.2\",\"queries\":{\"something\":{\"query\":\"select * from other;\",\"interval\":1200,\"description\":\"Check other.\"}}}"
  },
  "glob_pack_names": ["globpack"]
}
`
	buff := bytes.NewBufferString(testJSON)
	req, err := http.NewRequest("POST", r.server.URL+"/api/v1/kolide/osquery/config/import", buff)
	require.Nil(t, err)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", r.adminToken))
	client := &http.Client{}
	resp, err := client.Do(req)
	require.Nil(t, err)
	var impResponse importResponse
	err = json.NewDecoder(resp.Body).Decode(&impResponse)
	require.Nil(t, err)
	assert.Equal(t, 4, impResponse.Response.ImportStatusBySection[kolide.PacksSection].ImportCount)
}

func testImportConfigWithInvalidPlatform(t *testing.T, r *testResource) {
	testJSON := `
{
  "config": "{\"options\":{\"host_identifier\":\"hostname\",\"schedule_splay_percent\":10},\"schedule\":{\"macosx_kextstat\":{\"query\":\"SELECT * FROM kernel_extensions;\",\"interval\":10},\"foobar\":{\"query\":\"SELECT foo, bar, pid FROM foobar_table;\",\"interval\":600}},\"packs\":{\"*\":\"/path/to/glob/*\",\"external_pack\":\"/path/to/external_pack.conf\",\"internal_pack\":{\"discovery\":[\"select pid from processes where name = 'foobar';\",\"select count(*) from users where username like 'www%';\"],\"platform\":\"foo\",\"version\":\"1.5.2\",\"queries\":{\"active_directory\":{\"query\":\"select * from ad_config;\",\"interval\":1200,\"description\":\"Check each user's active directory cached settings.\"}}}},\"decorators\":{\"load\":[\"SELECT version FROM osquery_info\",\"SELECT uuid AS host_uuid FROM system_info\"],\"always\":[\"SELECT user AS username FROM logged_in_users WHERE user <> '' ORDER BY time LIMIT 1;\"],\"interval\":{\"3600\":[\"SELECT total_seconds AS uptime FROM uptime;\"]}},\"glob\":[\"globpack\"],\"yara\":{\"signatures\":{\"sig_group_1\":[\"/Users/wxs/sigs/foo.sig\",\"/Users/wxs/sigs/bar.sig\"],\"sig_group_2\":[\"/Users/wxs/sigs/baz.sig\"]},\"file_paths\":{\"system_binaries\":[\"sig_group_1\"],\"tmp\":[\"sig_group_1\",\"sig_group_2\"]}},\"file_paths\":{\"system_binaries\":[\"/usr/bin/%\",\"/usr/sbin/%\"],\"tmp\":[\"/Users/%/tmp/%%\",\"/tmp/%\"]}}",
  "external_pack_configs": {
    "external_pack": "{\"discovery\":[\"select pid from processes where name = 'baz';\"],\"platform\":\"linux\",\"version\":\"1.5.2\",\"queries\":{\"something\":{\"query\":\"select * from something;\",\"interval\":1200,\"description\":\"Check something.\"}}}",
    "globpack": "{\"discovery\":[\"select pid from processes where name = 'zip';\"],\"platform\":\"linux\",\"version\":\"1.5.2\",\"queries\":{\"something\":{\"query\":\"select * from other;\",\"interval\":1200,\"description\":\"Check other.\"}}}"
  },
  "glob_pack_names": ["globpack"]
}
`
	buff := bytes.NewBufferString(testJSON)
	req, err := http.NewRequest("POST", r.server.URL+"/api/v1/kolide/osquery/config/import", buff)
	require.Nil(t, err)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", r.adminToken))
	client := &http.Client{}
	resp, err := client.Do(req)
	require.Nil(t, err)
	var v mockValidationError
	err = json.NewDecoder(resp.Body).Decode(&v)
	require.Nil(t, err)
	require.Len(t, v.Errors, 1)
	assert.Equal(t, "'foo' is not a valid platform", v.Errors[0].Reason)
}

func testImportConfigWithMissingGlob(t *testing.T, r *testResource) {
	testJSON := `
  {
    "config": "{\"options\":{\"host_identifier\":\"hostname\",\"schedule_splay_percent\":10},\"schedule\":{\"macosx_kextstat\":{\"query\":\"SELECT * FROM kernel_extensions;\",\"interval\":10},\"foobar\":{\"query\":\"SELECT foo, bar, pid FROM foobar_table;\",\"interval\":600}},\"packs\":{\"*\":\"/path/to/glob/*\",\"external_pack\":\"/path/to/external_pack.conf\",\"internal_pack\":{\"discovery\":[\"select pid from processes where name = 'foobar';\",\"select count(*) from users where username like 'www%';\"],\"platform\":\"linux\",\"version\":\"1.5.2\",\"queries\":{\"active_directory\":{\"query\":\"select * from ad_config;\",\"interval\":1200,\"description\":\"Check each user's active directory cached settings.\"}}}},\"decorators\":{\"load\":[\"SELECT version FROM osquery_info\",\"SELECT uuid AS host_uuid FROM system_info\"],\"always\":[\"SELECT user AS username FROM logged_in_users WHERE user <> '' ORDER BY time LIMIT 1;\"],\"interval\":{\"3600\":[\"SELECT total_seconds AS uptime FROM uptime;\"]}},\"yara\":{\"signatures\":{\"sig_group_1\":[\"/Users/wxs/sigs/foo.sig\",\"/Users/wxs/sigs/bar.sig\"],\"sig_group_2\":[\"/Users/wxs/sigs/baz.sig\"]},\"file_paths\":{\"system_binaries\":[\"sig_group_1\"],\"tmp\":[\"sig_group_1\",\"sig_group_2\"]}},\"file_paths\":{\"system_binaries\":[\"/usr/bin/%\",\"/usr/sbin/%\"],\"tmp\":[\"/Users/%/tmp/%%\",\"/tmp/%\"]}}",
    "external_pack_configs": {
      "external_pack": "{\"discovery\":[\"select pid from processes where name = 'baz';\"],\"platform\":\"linux\",\"version\":\"1.5.2\",\"queries\":{\"something\":{\"query\":\"select * from something;\",\"interval\":1200,\"description\":\"Check something.\"}}}"
    }
  }
  `
	buff := bytes.NewBufferString(testJSON)
	req, err := http.NewRequest("POST", r.server.URL+"/api/v1/kolide/osquery/config/import", buff)
	require.Nil(t, err)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", r.adminToken))
	client := &http.Client{}
	resp, err := client.Do(req)
	require.Nil(t, err)
	var v mockValidationError
	err = json.NewDecoder(resp.Body).Decode(&v)
	require.Nil(t, err)
	require.Len(t, v.Errors, 1)
	assert.Equal(t, "missing glob packs", v.Errors[0].Reason)

}

func testImportConfigWithIntAsString(t *testing.T, r *testResource) {

	testJSON := `
  {
    "config": "{\"options\":{\"host_identifier\":\"hostname\",\"schedule_splay_percent\":10},\"schedule\":{\"macosx_kextstat\":{\"query\":\"SELECT * FROM kernel_extensions;\",\"interval\":\"10\"},\"foobar\":{\"query\":\"SELECT foo, bar, pid FROM foobar_table;\",\"interval\":600}},\"packs\":{\"external_pack\":\"/path/to/external_pack.conf\",\"internal_pack\":{\"discovery\":[\"select pid from processes where name = 'foobar';\",\"select count(*) from users where username like 'www%';\"],\"platform\":\"linux\",\"version\":\"1.5.2\",\"queries\":{\"active_directory\":{\"query\":\"select * from ad_config;\",\"interval\":\"1200\",\"description\":\"Check each user's active directory cached settings.\"}}}},\"decorators\":{\"load\":[\"SELECT version FROM osquery_info\",\"SELECT uuid AS host_uuid FROM system_info\"],\"always\":[\"SELECT user AS username FROM logged_in_users WHERE user <> '' ORDER BY time LIMIT 1;\"],\"interval\":{\"3600\":[\"SELECT total_seconds AS uptime FROM uptime;\"]}},\"yara\":{\"signatures\":{\"sig_group_1\":[\"/Users/wxs/sigs/foo.sig\",\"/Users/wxs/sigs/bar.sig\"],\"sig_group_2\":[\"/Users/wxs/sigs/baz.sig\"]},\"file_paths\":{\"system_binaries\":[\"sig_group_1\"],\"tmp\":[\"sig_group_1\",\"sig_group_2\"]}},\"file_paths\":{\"system_binaries\":[\"/usr/bin/%\",\"/usr/sbin/%\"],\"tmp\":[\"/Users/%/tmp/%%\",\"/tmp/%\"]}}",
    "external_pack_configs": {
      "external_pack": "{\"discovery\":[\"select pid from processes where name = 'baz';\"],\"platform\":\"linux\",\"version\":\"1.5.2\",\"queries\":{\"something\":{\"query\":\"select * from something;\",\"interval\":1200,\"description\":\"Check something.\"}}}"
    }
  }
  `
	buff := bytes.NewBufferString(testJSON)
	req, err := http.NewRequest("POST", r.server.URL+"/api/v1/kolide/osquery/config/import", buff)
	require.Nil(t, err)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", r.adminToken))
	client := &http.Client{}
	resp, err := client.Do(req)
	require.Nil(t, err)
	var impResponse importResponse
	err = json.NewDecoder(resp.Body).Decode(&impResponse)
	require.Nil(t, err)
	assert.Equal(t, 2, impResponse.Response.ImportStatusBySection[kolide.YARASigSection].ImportCount)
	assert.Equal(t, 4, impResponse.Response.ImportStatusBySection[kolide.DecoratorsSection].ImportCount)
}

func testImportConfig(t *testing.T, r *testResource) {

	testJSON := `
  {
    "config": "{\"options\":{\"host_identifier\":\"hostname\",\"schedule_splay_percent\":10},\"schedule\":{\"macosx_kextstat\":{\"query\":\"SELECT * FROM kernel_extensions;\",\"interval\":10},\"foobar\":{\"query\":\"SELECT foo, bar, pid FROM foobar_table;\",\"interval\":600}},\"packs\":{\"external_pack\":\"/path/to/external_pack.conf\",\"internal_pack\":{\"discovery\":[\"select pid from processes where name = 'foobar';\",\"select count(*) from users where username like 'www%';\"],\"platform\":\"linux\",\"version\":\"1.5.2\",\"queries\":{\"active_directory\":{\"query\":\"select * from ad_config;\",\"interval\":1200,\"description\":\"Check each user's active directory cached settings.\"}}}},\"decorators\":{\"load\":[\"SELECT version FROM osquery_info\",\"SELECT uuid AS host_uuid FROM system_info\"],\"always\":[\"SELECT user AS username FROM logged_in_users WHERE user <> '' ORDER BY time LIMIT 1;\"],\"interval\":{\"3600\":[\"SELECT total_seconds AS uptime FROM uptime;\"]}},\"yara\":{\"signatures\":{\"sig_group_1\":[\"/Users/wxs/sigs/foo.sig\",\"/Users/wxs/sigs/bar.sig\"],\"sig_group_2\":[\"/Users/wxs/sigs/baz.sig\"]},\"file_paths\":{\"system_binaries\":[\"sig_group_1\"],\"tmp\":[\"sig_group_1\",\"sig_group_2\"]}},\"file_paths\":{\"system_binaries\":[\"/usr/bin/%\",\"/usr/sbin/%\"],\"tmp\":[\"/Users/%/tmp/%%\",\"/tmp/%\"]}}",
    "external_pack_configs": {
      "external_pack": "{\"discovery\":[\"select pid from processes where name = 'baz';\"],\"platform\":\"linux\",\"version\":\"1.5.2\",\"queries\":{\"something\":{\"query\":\"select * from something;\",\"interval\":1200,\"description\":\"Check something.\"}}}"
    }
  }
  `
	buff := bytes.NewBufferString(testJSON)
	req, err := http.NewRequest("POST", r.server.URL+"/api/v1/kolide/osquery/config/import", buff)
	require.Nil(t, err)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", r.adminToken))
	client := &http.Client{}
	resp, err := client.Do(req)
	require.Nil(t, err)
	var impResponse importResponse
	err = json.NewDecoder(resp.Body).Decode(&impResponse)
	require.Nil(t, err)
	assert.Equal(t, 2, impResponse.Response.ImportStatusBySection[kolide.YARASigSection].ImportCount)
	assert.Equal(t, 4, impResponse.Response.ImportStatusBySection[kolide.DecoratorsSection].ImportCount)
}

func testImportConfigMissingExternal(t *testing.T, r *testResource) {
	testJSON := `
  {
    "config": "{\"options\":{\"host_identifier\":\"hostname\",\"schedule_splay_percent\":10},\"schedule\":{\"macosx_kextstat\":{\"query\":\"SELECT * FROM kernel_extensions;\",\"interval\":10},\"foobar\":{\"query\":\"SELECT foo, bar, pid FROM foobar_table;\",\"interval\":600}},\"packs\":{\"external_pack\":\"/path/to/external_pack.conf\",\"internal_pack\":{\"discovery\":[\"select pid from processes where name = 'foobar';\",\"select count(*) from users where username like 'www%';\"],\"platform\":\"linux\",\"version\":\"1.5.2\",\"queries\":{\"active_directory\":{\"query\":\"select * from ad_config;\",\"interval\":1200,\"description\":\"Check each user's active directory cached settings.\"}}}},\"decorators\":{\"load\":[\"SELECT version FROM osquery_info\",\"SELECT uuid AS host_uuid FROM system_info\"],\"always\":[\"SELECT user AS username FROM logged_in_users WHERE user <> '' ORDER BY time LIMIT 1;\"],\"interval\":{\"3603\":[\"SELECT total_seconds AS uptime FROM uptime;\"]}},\"yara\":{\"signatures\":{\"sig_group_1\":[\"/Users/wxs/sigs/foo.sig\",\"/Users/wxs/sigs/bar.sig\"],\"sig_group_2\":[\"/Users/wxs/sigs/baz.sig\"]},\"file_paths\":{\"system_binaries\":[\"sig_group_1\"],\"tmp\":[\"sig_group_1\",\"sig_group_2\"]}},\"file_paths\":{\"system_binaries\":[\"/usr/bin/%\",\"/usr/sbin/%\"],\"tmp\":[\"/Users/%/tmp/%%\",\"/tmp/%\"]}}"
  }
  `
	buff := bytes.NewBufferString(testJSON)
	req, err := http.NewRequest("POST", r.server.URL+"/api/v1/kolide/osquery/config/import", buff)
	require.Nil(t, err)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", r.adminToken))
	client := &http.Client{}
	resp, err := client.Do(req)
	require.Nil(t, err)

	var v mockValidationError
	err = json.NewDecoder(resp.Body).Decode(&v)
	require.Nil(t, err)
	require.Len(t, v.Errors, 2)
	assert.Equal(t, "missing content for 'external_pack'", v.Errors[0].Reason)
	assert.Equal(t, "interval '3603' must be divisible by 60", v.Errors[1].Reason)

}
