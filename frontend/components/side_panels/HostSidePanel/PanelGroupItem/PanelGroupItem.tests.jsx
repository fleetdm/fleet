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

  const labelComponent = mount(
    <PanelGroupItem item={validPanelGroupItem} />
  );

  const platformComponent = mount(
    <PanelGroupItem item={validPanelGroupItem} type="platform" />
  );

  it('renders the appropriate icon', () => {
    expect(labelComponent.find('PlatformIcon').length).toEqual(0);
    expect(labelComponent.find('Icon').length).toEqual(1);

    expect(platformComponent.find('PlatformIcon').length).toEqual(1);
  });

  it('renders the item text', () => {
    expect(labelComponent.text()).toContain(validPanelGroupItem.display_text);
  });

  it('renders the item count', () => {
    expect(labelComponent.text()).toContain(validPanelGroupItem.count);
  });
});
