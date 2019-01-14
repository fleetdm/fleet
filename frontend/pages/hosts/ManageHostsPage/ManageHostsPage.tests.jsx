import React from 'react';
import expect, { spyOn, restoreSpies } from 'expect';
import { mount } from 'enzyme';
import { noop } from 'lodash';

import hostActions from 'redux/nodes/entities/hosts/actions';
import labelActions from 'redux/nodes/entities/labels/actions';
import ConnectedManageHostsPage, { ManageHostsPage } from 'pages/hosts/ManageHostsPage/ManageHostsPage';
import { connectedComponent, createAceSpy, reduxMockStore, stubbedOsqueryTable } from 'test/helpers';
import { hostStub } from 'test/stubs';
import * as manageHostsPageActions from 'redux/nodes/components/ManageHostsPage/actions';

const allHostsLabel = { id: 1, display_text: 'All Hosts', slug: 'all-hosts', type: 'all', count: 22 };
const windowsLabel = { id: 2, display_text: 'Windows', slug: 'windows', type: 'platform', count: 22 };
const offlineHost = { ...hostStub, id: 111, status: 'offline' };
const offlineHostsLabel = { id: 5, display_text: 'OFFLINE', slug: 'offline', status: 'offline', type: 'status', count: 1 };
const customLabel = { id: 6, display_text: 'Custom Label', slug: 'custom-label', type: 'custom', count: 3 };
const mockStore = reduxMockStore({
  app: { config: {} },
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
    loadingHosts: false,
    loadingLabels: false,
    selectedOsqueryTable: stubbedOsqueryTable,
    statusLabels: {},
  };

  beforeEach(() => {
    const spyResponse = () => Promise.resolve([]);

    spyOn(hostActions, 'loadAll')
      .andReturn(spyResponse);
    spyOn(labelActions, 'loadAll')
      .andReturn(spyResponse);
    spyOn(manageHostsPageActions, 'getStatusLabelCounts')
      .andReturn(spyResponse);
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

  describe('host filtering', () => {
    it('filters hosts', () => {
      const allHostsLabelPageNode = mount(
        <ManageHostsPage
          {...props}
          hosts={[hostStub, offlineHost]}
          selectedLabel={allHostsLabel}
        />
      ).instance();
      const offlineHostsLabelPageNode = mount(
        <ManageHostsPage
          {...props}
          hosts={[hostStub, offlineHost]}
          selectedLabel={offlineHostsLabel}
        />
      ).instance();

      expect(allHostsLabelPageNode.filterAllHosts([hostStub, offlineHost], allHostsLabel)).toEqual([hostStub, offlineHost]);
      expect(offlineHostsLabelPageNode.filterAllHosts([hostStub, offlineHost], offlineHostsLabel)).toEqual([offlineHost]);
    });
  });

  describe('Adding a new label', () => {
    beforeEach(() => createAceSpy());
    afterEach(restoreSpies);

    const ownProps = { location: { hash: '#new_label' }, params: {} };
    const component = connectedComponent(ConnectedManageHostsPage, { props: ownProps, mockStore });

    it('renders a LabelForm component', () => {
      const page = mount(component);

      expect(page.find('LabelForm').length).toEqual(1);
    });

    it('displays "New Label Query" as the query form header', () => {
      const page = mount(component);

      expect(page.find('LabelForm').text()).toInclude('New Label Query');
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

    it('Renders the default description if the selected label does not have a description', () => {
      const defaultDescription = 'No description available.';
      const noDescriptionLabel = { ...allHostsLabel, description: undefined };
      const pageProps = {
        ...props,
        selectedLabel: noDescriptionLabel,
      };

      const Page = mount(<ManageHostsPage {...pageProps} />);

      expect(Page.find('.manage-hosts__header').text())
        .toInclude(defaultDescription);
    });

    it('Renders the label description if the selected label has a description', () => {
      const defaultDescription = 'No description available.';
      const labelDescription = 'This is the label description';
      const noDescriptionLabel = { ...allHostsLabel, description: labelDescription };
      const pageProps = {
        ...props,
        selectedLabel: noDescriptionLabel,
      };

      const Page = mount(<ManageHostsPage {...pageProps} />);

      expect(Page.find('.manage-hosts__header').text())
        .toInclude(labelDescription);
      expect(Page.find('.manage-hosts__header').text())
        .toNotInclude(defaultDescription);
    });
  });

  describe('Edit a label', () => {
    const ownProps = { location: {}, params: { active_label: 'custom-label' } };
    const Component = connectedComponent(ConnectedManageHostsPage, { props: ownProps, mockStore });

    it('renders the LabelForm when Edit is clicked', () => {
      const Page = mount(Component);
      const EditButton = Page
        .find('.manage-hosts__delete-label')
        .find('Button')
        .first();

      expect(Page.find('LabelForm').length).toEqual(0, 'Expected the LabelForm to not be on the page');

      EditButton.simulate('click');

      const LabelForm = Page.find('LabelForm');

      expect(LabelForm.length).toEqual(1, 'Expected the LabelForm to be on the page');

      expect(LabelForm.prop('formData')).toEqual(customLabel);
      expect(LabelForm.prop('isEdit')).toEqual(true);
    });
  });

  describe('Delete a label', () => {
    it('Deleted label after confirmation modal', () => {
      const ownProps = { location: {}, params: { active_label: 'custom-label' } };
      const component = connectedComponent(ConnectedManageHostsPage, {
        props: ownProps,
        mockStore,
      });
      const page = mount(component);
      const deleteBtn = page
        .find('.manage-hosts__delete-label')
        .find('Button')
        .last();

      spyOn(labelActions, 'destroy').andReturn((dispatch) => {
        dispatch({ type: 'labels_LOAD_REQUEST' });

        return Promise.resolve();
      });

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
      const deleteBtn = page.find('HostDetails').last().find('Button');

      spyOn(hostActions, 'destroy').andReturn((dispatch) => {
        dispatch({ type: 'hosts_LOAD_REQUEST' });

        return Promise.resolve();
      });

      expect(page.find('Modal').length).toEqual(0);

      deleteBtn.simulate('click');

      const confirmModal = page.find('Modal');

      expect(confirmModal.length).toEqual(1);

      const confirmBtn = confirmModal.find('.button--alert');
      confirmBtn.simulate('click');

      expect(hostActions.destroy).toHaveBeenCalledWith(offlineHost);
    });
  });

  describe('Add Host', () => {
    it('Open the Add Host modal from sidebar', () => {
      const page = mount(<ManageHostsPage {...props} hosts={[]} selectedLabel={allHostsLabel} />);
      const addNewHost = page.find('.host-side-panel__add-hosts');
      addNewHost.hostNodes().simulate('click');

      expect(page.find('AddHostModal').length).toBeGreaterThan(0);
    });

    it('Open the Add Host modal from Lonely Host', () => {
      const page = mount(<ManageHostsPage {...props} hosts={[]} selectedLabel={allHostsLabel} />);
      const addNewHost = page.find('LonelyHost').find('Button');
      addNewHost.simulate('click');

      expect(page.find('AddHostModal').length).toEqual(1);
    });
  });
});
