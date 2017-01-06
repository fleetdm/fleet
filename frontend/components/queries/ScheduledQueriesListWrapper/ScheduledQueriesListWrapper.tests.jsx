import React from 'react';
import expect, { createSpy, restoreSpies } from 'expect';
import { mount } from 'enzyme';

import { queryStub, scheduledQueryStub } from 'test/stubs';
import { fillInFormInput } from 'test/helpers';
import ScheduledQueriesListWrapper from './index';

const allQueries = [queryStub];
const scheduledQueries = [
  scheduledQueryStub,
  { ...scheduledQueryStub, id: 100, name: 'mac hosts' },
];

describe('ScheduledQueriesListWrapper - component', () => {
  afterEach(restoreSpies);

  it('renders the "Remove Query" button when queries have been selected', () => {
    const component = mount(
      <ScheduledQueriesListWrapper
        allQueries={allQueries}
        scheduledQueries={scheduledQueries}
      />
    );

    component.find('Checkbox').last().find('input').simulate('change');

    const addQueryBtn = component.find('Button').find({ children: 'Add New Query' });
    const removeQueryBtn = component.find('Button').find({ children: ['Remove ', 'Query'] });

    expect(addQueryBtn.length).toEqual(0);
    expect(removeQueryBtn.length).toEqual(1);
  });

  it('calls the onRemoveScheduledQueries prop', () => {
    const spy = createSpy();
    const component = mount(
      <ScheduledQueriesListWrapper
        allQueries={allQueries}
        onRemoveScheduledQueries={spy}
        scheduledQueries={[scheduledQueryStub]}
      />
    );

    component.find('Checkbox').last().find('input').simulate('change');

    const removeQueryBtn = component.find('Button').find({ children: ['Remove ', 'Query'] });

    removeQueryBtn.simulate('click');

    expect(spy).toHaveBeenCalledWith([scheduledQueryStub.id]);
  });

  it('filters queries', () => {
    const component = mount(
      <ScheduledQueriesListWrapper
        allQueries={allQueries}
        scheduledQueries={scheduledQueries}
      />
    );

    const searchQueriesInput = component.find({ name: 'search-queries' });
    const QueriesList = component.find('ScheduledQueriesList');

    expect(QueriesList.prop('scheduledQueries')).toEqual(scheduledQueries);

    fillInFormInput(searchQueriesInput, 'something that does not match');

    expect(QueriesList.prop('scheduledQueries')).toEqual([]);
  });

  it('allows selecting all scheduled queries at once', () => {
    const allScheduledQueryIDs = scheduledQueries.map(sq => sq.id);
    const component = mount(
      <ScheduledQueriesListWrapper
        allQueries={allQueries}
        scheduledQueries={scheduledQueries}
      />
    );
    const selectAllCheckbox = component.find({ name: 'select-all-scheduled-queries' });

    selectAllCheckbox.simulate('change');

    expect(component.state('selectedScheduledQueryIDs')).toEqual(allScheduledQueryIDs);

    selectAllCheckbox.simulate('change');

    expect(component.state('selectedScheduledQueryIDs')).toEqual([]);
  });
});
