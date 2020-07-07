import React from 'react';
import expect from 'expect';
import { mount } from 'enzyme';
import { noop } from 'lodash';

import { scheduledQueryStub } from 'test/stubs';
import ScheduledQueriesList from './index';

const scheduledQueries = [
  { ...scheduledQueryStub, id: 1 },
  { ...scheduledQueryStub, id: 2 },
];

describe('ScheduledQueriesList - component', () => {
  it('renders a ScheduledQueriesListItem for each scheduled query', () => {
    const component = mount(
      <ScheduledQueriesList
        allQueries={[]}
        onHidePackForm={noop}
        onSelectQuery={noop}
        scheduledQueries={scheduledQueries}
        selectedScheduledQueryIDs={[]}
      />,
    );

    expect(component.find('ScheduledQueriesListItem').length).toEqual(2);
  });

  it('renders "No queries found" help text when scheduled queries are available but the scheduled queries are filtered out', () => {
    const component = mount(
      <ScheduledQueriesList
        allQueries={[]}
        isScheduledQueriesAvailable
        onHidePackForm={noop}
        onSelectQuery={noop}
        scheduledQueries={[]}
        selectedScheduledQueryIDs={[]}
      />,
    );

    expect(component.text()).toInclude('No queries matched your search criteria');
  });

  it('renders initial help text when no queries have been scheduled yet', () => {
    const component = mount(
      <ScheduledQueriesList
        allQueries={[]}
        onHidePackForm={noop}
        onSelectQuery={noop}
        scheduledQueries={[]}
        selectedScheduledQueryIDs={[]}
      />,
    );

    expect(component.text()).toInclude("First let's add a query");
  });
});
