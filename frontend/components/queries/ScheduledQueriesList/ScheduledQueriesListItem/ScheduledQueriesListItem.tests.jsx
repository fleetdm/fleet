import React from 'react';
import expect, { createSpy, restoreSpies } from 'expect';
import { mount } from 'enzyme';
import { noop } from 'lodash';

import { scheduledQueryStub } from 'test/stubs';
import ScheduledQueriesListItem from './index';

const defaultProps = {
  checked: false,
  onCheck: noop,
  onSelect: noop,
  scheduledQuery: scheduledQueryStub,
};

describe('ScheduledQueriesListItem - component', () => {
  afterEach(restoreSpies);

  it('renders the scheduled query data', () => {
    const component = mount(<ScheduledQueriesListItem {...defaultProps} />);
    expect(component.text()).toInclude(scheduledQueryStub.name);
    expect(component.text()).toInclude(scheduledQueryStub.interval);
    expect(component.text()).toInclude(scheduledQueryStub.shard);
    expect(component.find('PlatformIcon').length).toEqual(1);
  });

  it('renders a Checkbox component', () => {
    const component = mount(<ScheduledQueriesListItem {...defaultProps} />);
    expect(component.find('Checkbox').length).toEqual(1);
  });

  it('calls the onCheck prop when a checkbox is changed', () => {
    const onCheckSpy = createSpy();
    const props = { ...defaultProps, onCheck: onCheckSpy };
    const component = mount(<ScheduledQueriesListItem {...props} />);
    const checkbox = component.find('Checkbox').first();

    checkbox.find('input').simulate('change');

    expect(onCheckSpy).toHaveBeenCalledWith(true, scheduledQueryStub.id);
  });

  it('calls the onSelect prop when a list item is selected', () => {
    const spy = createSpy();
    const props = { ...defaultProps, onSelect: spy };
    const component = mount(<ScheduledQueriesListItem {...props} />);
    const tableRow = component.find('ClickableTableRow');

    tableRow.simulate('click');

    expect(spy).toHaveBeenCalledWith(scheduledQueryStub);
  });
});
