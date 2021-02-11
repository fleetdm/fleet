import React from 'react';
import { noop } from 'lodash';
import { shallow, mount } from 'enzyme';

import { hostStub } from 'test/stubs';
import HostContainer from './HostContainer';

const allHostsLabel = { id: 1, display_text: 'All Hosts', slug: 'all-hosts', type: 'all', count: 22 };
const customLabel = { id: 6, display_text: 'Custom Label', slug: 'custom-label', type: 'custom', count: 3 };

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

  it('renders Spinner while hosts are loading', () => {
    const loadingProps = { ...props, loadingHosts: true };
    const page = mount(<HostContainer {...loadingProps} hosts={[]} selectedLabel={allHostsLabel} />);

    expect(page.find('Spinner').length).toEqual(1);
  });

  it('displays getting started text if no hosts available', () => {
    const page = mount(<HostContainer {...props} hosts={[]} selectedLabel={allHostsLabel} />);

    expect(page.find('h2').text()).toEqual('Get started adding hosts to Fleet.');
  });

  it('renders message if no hosts available and not on All Hosts', () => {
    const page = mount(<HostContainer {...props} hosts={[]} selectedLabel={customLabel} />);

    expect(page.find('.host-container--no-hosts').length).toEqual(1);
  });

  it('renders the HostsDataTable if there are hosts', () => {
    const page = shallow(<HostContainer {...props} />);

    expect(page.find('HostsDataTable').length).toEqual(1);
  });

  it('does not render sidebar if labels are loading', () => {
    const loadingProps = { ...props, loadingLabels: true };
    const page = mount(<HostContainer {...loadingProps} hosts={[]} selectedLabel={allHostsLabel} />);

    expect(page.find('HostSidePanel').length).toEqual(0);
  });
});
