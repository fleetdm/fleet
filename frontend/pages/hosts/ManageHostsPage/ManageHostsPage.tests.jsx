import React from 'react';
import expect, { spyOn, restoreSpies } from 'expect';
import { mount } from 'enzyme';
import { noop } from 'lodash';

import hostActions from 'redux/nodes/entities/hosts/actions';
import labelActions from 'redux/nodes/entities/labels/actions';
import ConnectedManageHostsPage, { ManageHostsPage } from 'pages/hosts/ManageHostsPage/ManageHostsPage';
import { connectedComponent, createAceSpy, reduxMockStore, stubbedOsqueryTable } from 'test/helpers';
import { hostStub } from 'test/stubs';

const allHostsLabel = { id: 1, display_text: 'All Hosts', slug: 'all-hosts', type: 'all', count: 22 };
const windowsLabel = { id: 2, display_text: 'Windows', slug: 'windows', type: 'platform', count: 22 };
const offlineHost = { ...hostStub, id: 111, status: 'offline' };
const offlineHostsLabel = { id: 5, display_text: 'OFFLINE', slug: 'offline', status: 'offline', type: 'status', count: 1 };
const customLabel = { id: 6, display_text: 'Custom Label', slug: 'custom-label', type: 'custom', count: 3 };
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
    hosts: {
      data: {
        [hostStub.id]: hostStub,
        [offlineHost.id]: offlineHost,
      },
    },
    labels: {
      data: {
        1: allHostsLabel,
        2: windowsLabel,
        3: { id: 3, display_text: 'Ubuntu', slug: 'ubuntu', type: 'platform', count: 22 },
        4: { id: 4, display_text: 'ONLINE', slug: 'online', type: 'status', count: 22 },
        5: offlineHostsLabel,
        6: customLabel,
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
    statusLabels: {},
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

  describe('header', () => {
    it('displays "1 Host Total" when there is 1 host', () => {
      const oneHostLabel = { ...allHostsLabel, count: 1 };
      const page = mount(<ManageHostsPage {...props} selectedLabel={oneHostLabel} />);

      expect(page.text()).toInclude('1 Host Total');
    });

    it('displays "#{count} Hosts Total" when there are more than 1 host', () => {
      const oneHostLabel = { ...allHostsLabel, count: 2 };
      const page = mount(<ManageHostsPage {...props} selectedLabel={oneHostLabel} />);

      expect(page.text()).toInclude('2 Hosts Total');
    });
  });

  describe('host rendering', () => {
    it('render LonelyHost if no hosts available', () => {
      const page = mount(<ManageHostsPage {...props} hosts={[]} selectedLabel={allHostsLabel} />);

      expect(page.find('LonelyHost').length).toEqual(1);
    });

    it('renders message if no hosts available and not on All Hosts', () => {
      const page = mount(<ManageHostsPage {...props} hosts={[]} selectedLabel={customLabel} />);

      expect(page.find('.manage-hosts__no-hosts').length).toEqual(1);
    });

    it('renders hosts as HostDetails by default', () => {
      const page = mount(<ManageHostsPage {...props} hosts={[hostStub]} />);

      expect(page.find('HostDetails').length).toEqual(1);
    });

    it('renders hosts as HostsTable when the display is "List"', () => {
      const page = mount(<ManageHostsPage {...props} display="List" hosts={[hostStub]} />);

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

    it('filters hosts', () => {
      const allHostsLabelPageNode = mount(
        <ManageHostsPage
          {...props}
          hosts={[hostStub, offlineHost]}
          selectedLabel={allHostsLabel}
        />
      ).node;
      const offlineHostsLabelPageNode = mount(
        <ManageHostsPage
          {...props}
          hosts={[hostStub, offlineHost]}
          selectedLabel={offlineHostsLabel}
        />
      ).node;

      expect(allHostsLabelPageNode.filterHosts()).toEqual([hostStub, offlineHost]);
      expect(offlineHostsLabelPageNode.filterHosts()).toEqual([offlineHost]);
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

  describe('Delete a label', () => {
    it('Deleted label after confirmation modal', () => {
      const ownProps = { location: {}, params: { active_label: 'custom-label' } };
      const component = connectedComponent(ConnectedManageHostsPage, { props: ownProps, mockStore });
      const page = mount(component);
      const deleteBtn = page.find('.manage-hosts__delete-label').find('button');

      spyOn(labelActions, 'destroy').andCallThrough();

      expect(page.find('Modal').length).toEqual(0);

      deleteBtn.simulate('click');

      const confirmModal = page.find('Modal');

      expect(confirmModal.length).toEqual(1);

      const confirmBtn = confirmModal.find('.button--alert');
      confirmBtn.simulate('click');

      expect(labelActions.destroy).toHaveBeenCalledWith(customLabel);
    });
  });

  describe('Delete a host', () => {
    it('Deleted host after confirmation modal', () => {
      const ownProps = { location: {}, params: { active_label: 'all-hosts' } };
      const component = connectedComponent(ConnectedManageHostsPage, { props: ownProps, mockStore });
      const page = mount(component);
      const deleteBtn = page.find('HostDetails').first().find('Button');

      spyOn(hostActions, 'destroy').andCallThrough();

      expect(page.find('Modal').length).toEqual(0);

      deleteBtn.simulate('click');

      const confirmModal = page.find('Modal');

      expect(confirmModal.length).toEqual(1);

      const confirmBtn = confirmModal.find('.button--alert');
      confirmBtn.simulate('click');

      expect(hostActions.destroy).toHaveBeenCalledWith(hostStub);
    });
  });
});
