import React from 'react';
import { noop } from 'lodash';
import { shallow, mount } from 'enzyme';

import { hostStub } from 'test/stubs';
import HostContainer from './HostContainer';

const allHostsLabel = { id: 1, display_text: 'All Hosts', slug: 'all-hosts', type: 'all', count: 22 };

describe('HostsContainer - component', () => {
  const props = {
    hosts: [hostStub],
    selectedLabel: allHostsLabel,
    loadingHosts: false,
    displayType: 'Grid',
    toggleAddHostModal: noop,
    toggleDeleteHostModal: noop,
    onQueryHost: noop,
  };

  // TODO: come back and implement this again.
  it('displays getting started text if no hosts available', () => {
    const page = shallow(<HostContainer {...props} hosts={[]} selectedLabel={allHostsLabel} />);

    expect(page.find('h2').text()).toEqual('Get started adding hosts to Fleet.');
  });

  it('renders the HostsDataTable if there are hosts', () => {
    const page = shallow(<HostContainer {...props} />);

    expect(page.find('HostsDataTable').length).toEqual(1);
  });
});
