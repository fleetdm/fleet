import React from 'react';
import expect from 'expect';
import { mount } from 'enzyme';

import PanelGroup from './PanelGroup';

describe('PanelGroup - component', () => {
  const validPanelGroupItems = [
    { type: 'all', title: 'All Hosts', hosts_count: 20 },
    { type: 'platform', title: 'MAC OS', hosts_count: 10 },
    { type: 'status', title: 'ONLINE', hosts_count: 10 },
  ];

  const component = mount(
    <PanelGroup groupItems={validPanelGroupItems} />
  );

  it('renders a PanelGroupItem for each group item', () => {
    const panelGroupItems = component.find('PanelGroupItem');

    expect(panelGroupItems.length).toEqual(3);
  });
});

