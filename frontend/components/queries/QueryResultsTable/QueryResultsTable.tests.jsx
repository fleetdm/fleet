import React from 'react';
import expect from 'expect';
import { keys } from 'lodash';
import { mount } from 'enzyme';

import QueryResultsTable from 'components/queries/QueryResultsTable';

const host = {
  detail_updated_at: '2016-10-25T16:24:27.679472917-04:00',
  hostname: 'jmeller-mbp.local',
  id: 1,
  ip: '192.168.1.10',
  mac: '10:11:12:13:14:15',
  memory: 4145483776,
  os_version: 'Mac OS X 10.11.6',
  osquery_version: '2.0.0',
  platform: 'darwin',
  status: 'online',
  updated_at: '0001-01-01T00:00:00Z',
  uptime: 3600000000000,
  uuid: '1234-5678-9101',
};
const queryResult = {
  distributed_query_execution_id: 4,
  host,
  rows: [{ cwd: '/' }],
};

const campaignWithNoQueryResults = {
  created_at: '0001-01-01T00:00:00Z',
  deleted: false,
  deleted_at: null,
  id: 4,
  query_id: 12,
  status: 0,
  totals: {
    count: 3,
    online: 2,
  },
  updated_at: '0001-01-01T00:00:00Z',
  user_id: 1,
};
const campaignWithQueryResults = {
  ...campaignWithNoQueryResults,
  query_results: [
    { hostname: host.hostname, cwd: '/' },
  ],
};

describe('QueryResultsTable - component', () => {
  const componentWithoutQueryResults = mount(
    <QueryResultsTable campaign={campaignWithNoQueryResults} />
  );
  const componentWithQueryResults = mount(
    <QueryResultsTable campaign={campaignWithQueryResults} />
  );

  it('renders', () => {
    expect(componentWithoutQueryResults.length).toEqual(1);
    expect(componentWithQueryResults.length).toEqual(1);
  });

  it('does not return HTML when there are no query results', () => {
    expect(componentWithoutQueryResults.html()).toNotExist();
  });

  it('renders a ProgressBar component', () => {
    expect(
      componentWithQueryResults.find('ProgressBar').length
    ).toEqual(1);
  });

  it('sets the column headers to the keys of the query results', () => {
    const queryResultKeys = keys(queryResult.rows[0]);
    const tableHeaderText = componentWithQueryResults.find('thead').text();

    queryResultKeys.forEach((key) => {
      expect(tableHeaderText).toInclude(key);
    });
  });
});
