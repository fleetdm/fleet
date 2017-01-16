import React from 'react';
import expect, { restoreSpies } from 'expect';
import { mount } from 'enzyme';
import { noop } from 'lodash';

import ConnectedManageHostsPage, { ManageHostsPage } from 'pages/hosts/ManageHostsPage/ManageHostsPage';
import { connectedComponent, createAceSpy, reduxMockStore, stubbedOsqueryTable } from 'test/helpers';

const host = {
  detail_updated_at: '2016-10-25T16:24:27.679472917-04:00',
  hostname: 'jmeller-mbp.local',
  id: 1,
  ip: '192.168.1.10',
  mac: '10:11:12:13:14:15',
  memory: 4145483776,
  os_version: 'Mac OS X 10.11.6',
  osquery_version: '2.0.0',
  platform: 'darwin',
  status: 'online',
  updated_at: '0001-01-01T00:00:00Z',
  uptime: 3600000000000,
  uuid: '1234-5678-9101',
};
const allHostsLabel = { id: 1, display_text: 'All Hosts', slug: 'all-hosts', type: 'all', count: 22 };
const windowsLabel = { id: 2, display_text: 'Windows', slug: 'windows', type: 'platform', count: 22 };
const mockStore = reduxMockStore({
  components: {
    ManageHostsPage: {
      display: 'Grid',
      selectedLabel: { id: 100, display_text: 'All Hosts', type: 'all', count: 22 },
      status_labels: {},
    },
    QueryPages: {
      selectedOsqueryTable: stubbedOsqueryTable,
    },
  },
  entities: {
    labels: {
      data: {
        1: allHostsLabel,
        2: windowsLabel,
        3: { id: 3, display_text: 'Ubuntu', slug: 'ubuntu', type: 'platform', count: 22 },
        4: { id: 4, display_text: 'ONLINE', slug: 'online', type: 'status', count: 22 },
      },
    },
  },
});

describe('ManageHostsPage - component', () => {
  const props = {
    dispatch: noop,
    hosts: [],
    labels: [],
    selectedOsqueryTable: stubbedOsqueryTable,
  };

  beforeEach(() => {
    createAceSpy();
  });
  afterEach(restoreSpies);

  describe('side panels', () => {
    it('renders a HostSidePanel when not adding a new label', () => {
      const page = mount(<ManageHostsPage {...props} />);

      expect(page.find('HostSidePanel').length).toEqual(1);
    });

    it('renders a QuerySidePanel when adding a new label', () => {
      const ownProps = { location: { hash: '#new_label' }, params: {} };
      const component = connectedComponent(ConnectedManageHostsPage, { props: ownProps, mockStore });
      const page = mount(component);

      expect(page.find('QuerySidePanel').length).toEqual(1);
    });
  });

  describe('host rendering', () => {
    it('renders hosts as HostDetails by default', () => {
      const page = mount(<ManageHostsPage {...props} hosts={[host]} />);

      expect(page.find('HostDetails').length).toEqual(1);
    });

    it('renders hosts as HostsTable when the display is "List"', () => {
      const page = mount(<ManageHostsPage {...props} display="List" hosts={[host]} />);

      expect(page.find('HostsTable').length).toEqual(1);
    });

    it('toggles between displays', () => {
      const ownProps = { location: {}, params: {} };
      const component = connectedComponent(ConnectedManageHostsPage, { props: ownProps, mockStore });
      const page = mount(component);
      const button = page.find('Rocker').find('button');
      const toggleDisplayAction = {
        type: 'SET_DISPLAY',
        payload: {
          display: 'List',
        },
      };

      button.simulate('click');

      expect(mockStore.getActions()).toInclude(toggleDisplayAction);
    });
  });

  describe('Adding a new label', () => {
    beforeEach(() => createAceSpy());
    afterEach(restoreSpies);

    const ownProps = { location: { hash: '#new_label' }, params: {} };
    const component = connectedComponent(ConnectedManageHostsPage, { props: ownProps, mockStore });

    it('renders a QueryForm component', () => {
      const page = mount(component);

      expect(page.find('QueryForm').length).toEqual(1);
    });

    it('displays "New Label Query" as the query form header', () => {
      const page = mount(component);

      expect(page.find('QueryForm').text()).toInclude('New Label Query');
    });
  });

  describe('Active label', () => {
    beforeEach(() => createAceSpy());
    afterEach(restoreSpies);

    it('Displays the all hosts label as the active label by default', () => {
      const ownProps = { location: {}, params: {} };
      const component = connectedComponent(ConnectedManageHostsPage, { props: ownProps, mockStore });
      const page = mount(component);

      expect(page.find('HostSidePanel').props()).toInclude({
        selectedLabel: allHostsLabel,
      });
    });

    it('Displays the windows label as the active label', () => {
      const ownProps = { location: {}, params: { active_label: 'windows' } };
      const component = connectedComponent(ConnectedManageHostsPage, { props: ownProps, mockStore });
      const page = mount(component);

      expect(page.find('HostSidePanel').props()).toInclude({
        selectedLabel: windowsLabel,
      });
    });
  });
});
