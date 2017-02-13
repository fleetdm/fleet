import React from 'react';
import expect, { spyOn, restoreSpies } from 'expect';
import { mount } from 'enzyme';

import { connectedComponent, reduxMockStore } from 'test/helpers';
import { packStub } from 'test/stubs';
import ConnectedEditPackPage, { EditPackPage } from 'pages/packs/EditPackPage/EditPackPage';
import hostActions from 'redux/nodes/entities/hosts/actions';
import labelActions from 'redux/nodes/entities/labels/actions';
import packActions from 'redux/nodes/entities/packs/actions';
import queryActions from 'redux/nodes/entities/queries/actions';
import scheduledQueryActions from 'redux/nodes/entities/scheduled_queries/actions';

describe('EditPackPage - component', () => {
  beforeEach(() => {
    const spyResponse = () => Promise.resolve([]);

    spyOn(hostActions, 'loadAll').andReturn(spyResponse);
    spyOn(labelActions, 'loadAll').andReturn(spyResponse);
    spyOn(packActions, 'load').andReturn(spyResponse);
    spyOn(queryActions, 'loadAll').andReturn(spyResponse);
    spyOn(scheduledQueryActions, 'loadAll').andReturn(spyResponse);
  });

  afterEach(restoreSpies);

  const store = {
    entities: {
      hosts: { loading: false, data: {} },
      labels: { loading: false, data: {} },
      packs: {
        loading: false,
        data: {
          [packStub.id]: packStub,
        },
      },
      scheduled_queries: { loading: false, data: {} },
    },
  };
  const page = mount(connectedComponent(ConnectedEditPackPage, {
    props: { params: { id: String(packStub.id) }, route: {} },
    mockStore: reduxMockStore(store),
  }));

  describe('rendering', () => {
    it('does not render when packs are loading', () => {
      const packsLoadingStore = {
        entities: {
          ...store.entities,
          packs: { ...store.entities.packs, loading: true },
        },
      };

      const loadingPacksPage = mount(connectedComponent(ConnectedEditPackPage, {
        props: { params: { id: String(packStub.id) }, route: {} },
        mockStore: reduxMockStore(packsLoadingStore),
      }));

      expect(loadingPacksPage.html()).toNotExist();
    });

    it('does not render when scheduled queries are loading', () => {
      const scheduledQueriesLoadingStore = {
        entities: {
          ...store.entities,
          scheduled_queries: { ...store.entities.scheduled_queries, loading: true },
        },
      };

      const loadingScheduledQueriesPage = mount(connectedComponent(ConnectedEditPackPage, {
        props: { params: { id: String(packStub.id) }, route: {} },
        mockStore: reduxMockStore(scheduledQueriesLoadingStore),
      }));

      expect(loadingScheduledQueriesPage.html()).toNotExist();
    });

    it('does not render when there is no pack', () => {
      const noPackStore = {
        entities: {
          ...store.entities,
          packs: { data: {}, loading: false },
        },
      };

      const noPackPage = mount(connectedComponent(ConnectedEditPackPage, {
        props: { params: { id: String(packStub.id) }, route: {} },
        mockStore: reduxMockStore(noPackStore),
      }));

      expect(noPackPage.html()).toNotExist();
    });

    it('renders', () => {
      expect(page.length).toEqual(1);
    });

    it('renders a EditPackFormWrapper component', () => {
      expect(page.find('EditPackFormWrapper').length).toEqual(1);
    });

    it('renders a ScheduleQuerySidePanel component', () => {
      expect(page.find('ScheduleQuerySidePanel').length).toEqual(1);
    });
  });

  describe('updating a pack', () => {
    it('only sends the updated attributes to the server', () => {
      spyOn(packActions, 'update');
      const dispatch = () => Promise.resolve();

      const updatedAttrs = { name: 'Updated pack name' };
      const updatedPack = { ...packStub, ...updatedAttrs };
      const props = {
        allQueries: [],
        dispatch,
        isEdit: false,
        packHosts: [],
        packLabels: [],
        scheduledQueries: [],
      };

      const pageNode = mount(<EditPackPage {...props} pack={packStub} />).node;

      pageNode.handlePackFormSubmit(updatedPack);

      expect(packActions.update).toHaveBeenCalledWith(packStub, updatedAttrs);
    });
  });
});
