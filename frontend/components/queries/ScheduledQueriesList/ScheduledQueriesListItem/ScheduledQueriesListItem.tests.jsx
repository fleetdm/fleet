import React from 'react';
import expect, { createSpy, restoreSpies } from 'expect';
import { mount } from 'enzyme';
import { noop } from 'lodash';

import { scheduledQueryStub } from 'test/stubs';
import ScheduledQueriesListItem from './index';

describe('ScheduledQueriesListItem - component', () => {
  afterEach(restoreSpies);

  it('renders the scheduled query data', () => {
    const component = mount(<ScheduledQueriesListItem checked={false} onSelect={noop} scheduledQuery={scheduledQueryStub} />);
    expect(component.text()).toInclude(scheduledQueryStub.name);
    expect(component.text()).toInclude(scheduledQueryStub.interval);
    expect(component.find('PlatformIcon').length).toEqual(1);
  });

  it('renders a Checkbox component', () => {
    const component = mount(<ScheduledQueriesListItem checked={false} onSelect={noop} scheduledQuery={scheduledQueryStub} />);
    expect(component.find('Checkbox').length).toEqual(1);
  });

  it('calls the onSelect prop when a checkbox is changed', () => {
    const onSelectSpy = createSpy();
    const component = mount(<ScheduledQueriesListItem checked={false} onSelect={onSelectSpy} scheduledQuery={scheduledQueryStub} />);
    const checkbox = component.find('Checkbox').first();

    checkbox.find('input').simulate('change');

    expect(onSelectSpy).toHaveBeenCalledWith(true, scheduledQueryStub.id);
  });
});
