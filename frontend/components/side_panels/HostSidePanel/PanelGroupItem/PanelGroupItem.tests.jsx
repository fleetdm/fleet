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
  const validStatusGroupItem = {
    count: 111,
    display_text: 'Online Hosts',
    id: 'online',
    type: 'status',
  };
  const statusLabels = {
    online_count: 20,
    loading_counts: false,
  };
  const loadingStatusLabels = {
    online_count: 20,
    loading_counts: true,
  };

  const labelComponent = mount(
    <PanelGroupItem item={validPanelGroupItem} statusLabels={statusLabels} />
  );

  const platformComponent = mount(
    <PanelGroupItem item={validPanelGroupItem} statusLabels={statusLabels} type="platform" />
  );

  const statusLabelComponent = mount(
    <PanelGroupItem item={validStatusGroupItem} statusLabels={statusLabels} type="status" />
  );

  const loadingStatusLabelComponent = mount(
    <PanelGroupItem item={validStatusGroupItem} statusLabels={loadingStatusLabels} type="status" />
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
    expect(statusLabelComponent.text()).toNotContain(validStatusGroupItem.count);
    expect(statusLabelComponent.text()).toContain(statusLabels.online_count);
    expect(loadingStatusLabelComponent.text()).toNotContain(statusLabels.online_count);
    expect(loadingStatusLabelComponent.text()).toNotContain(validPanelGroupItem.count);
  });
});
