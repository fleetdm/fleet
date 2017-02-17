import React from 'react';
import expect, { createSpy, restoreSpies } from 'expect';
import { mount } from 'enzyme';
import { noop } from 'lodash';

import { scheduledQueryStub } from 'test/stubs';
import { fillInFormInput } from 'test/helpers';
import ScheduledQueriesListWrapper from './index';

const scheduledQueries = [
  scheduledQueryStub,
  { ...scheduledQueryStub, id: 100, name: 'mac hosts' },
];
const defaultProps = {
  onRemoveScheduledQueries: noop,
  onScheduledQueryFormSubmit: noop,
  onSelectScheduledQuery: noop,
  scheduledQueries,
};

describe('ScheduledQueriesListWrapper - component', () => {
  afterEach(restoreSpies);

  it('renders the "Remove Query" button when queries have been selected', () => {
    const component = mount(<ScheduledQueriesListWrapper {...defaultProps} />);

    component.find('Checkbox').last().find('input').simulate('change');

    const addQueryBtn = component.find('Button').find({ children: 'Add New Query' });
    const removeQueryBtn = component.find('Button').find({ children: ['Remove ', 'Query'] });

    expect(addQueryBtn.length).toEqual(0);
    expect(removeQueryBtn.length).toEqual(1);
  });

  it('calls the onRemoveScheduledQueries prop', () => {
    const spy = createSpy();
    const props = { ...defaultProps, onRemoveScheduledQueries: spy };
    const component = mount(<ScheduledQueriesListWrapper {...props} />);

    component
      .find('Checkbox')
      .find({ name: `scheduled-query-checkbox-${scheduledQueryStub.id}` })
      .find('input')
      .simulate('change');

    const removeQueryBtn = component.find('Button').find({ children: ['Remove ', 'Query'] });

    removeQueryBtn.simulate('click');

    expect(spy).toHaveBeenCalledWith([scheduledQueryStub.id]);
  });

  it('filters queries', () => {
    const component = mount(<ScheduledQueriesListWrapper {...defaultProps} />);

    const searchQueriesInput = component.find({ name: 'search-queries' });
    const QueriesList = component.find('ScheduledQueriesList');

    expect(QueriesList.prop('scheduledQueries')).toEqual(scheduledQueries);

    fillInFormInput(searchQueriesInput, 'something that does not match');

    expect(QueriesList.prop('scheduledQueries')).toEqual([]);
  });

  it('allows selecting all scheduled queries at once', () => {
    const allScheduledQueryIDs = scheduledQueries.map(sq => sq.id);
    const component = mount(<ScheduledQueriesListWrapper {...defaultProps} />);
    const selectAllCheckbox = component.find({ name: 'select-all-scheduled-queries' });

    selectAllCheckbox.simulate('change');

    expect(component.state('checkedScheduledQueryIDs')).toEqual(allScheduledQueryIDs);

    selectAllCheckbox.simulate('change');

    expect(component.state('checkedScheduledQueryIDs')).toEqual([]);
  });
});
