import React from 'react';
import expect from 'expect';
import { mount } from 'enzyme';

import ScheduleQuerySidePanel from './ScheduleQuerySidePanel';

describe('ScheduleQuerySidePanel - component', () => {
  const component = mount(<ScheduleQuerySidePanel />);

  it('renders SearchPackQuery', () => {
    const scheduleQuery = component.find('SearchPackQuery');

    expect(scheduleQuery.length).toEqual(1);
  });
});
