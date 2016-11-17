import React from 'react';
import expect from 'expect';
import { mount } from 'enzyme';

import PanelGroupItem from './PanelGroupItem';

describe('PanelGroupItem - component', () => {
  const validPanelGroupItem = {
    count: 20,
    display_text: 'All Hosts',
    type: 'all',
  };

  const component = mount(
    <PanelGroupItem item={validPanelGroupItem} />
  );

  it('renders the icon', () => {
    const icon = component.find('i.kolidecon-hosts');

    expect(icon.length).toEqual(1);
  });

  it('renders the item text', () => {
    expect(component.text()).toContain(validPanelGroupItem.display_text);
  });

  it('renders the item count', () => {
    expect(component.text()).toContain(validPanelGroupItem.count);
  });
});
