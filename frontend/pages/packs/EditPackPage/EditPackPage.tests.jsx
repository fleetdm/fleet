import React from 'react';
import expect, { spyOn, restoreSpies } from 'expect';
import { mount } from 'enzyme';

import { connectedComponent, reduxMockStore } from 'test/helpers';
import { packStub } from 'test/stubs';
import ConnectedEditPackPage, { EditPackPage } from 'pages/packs/EditPackPage/EditPackPage';
import packActions from 'redux/nodes/entities/packs/actions';

describe('EditPackPage - component', () => {
  afterEach(restoreSpies);

  const store = {
    entities: {
      hosts: { data: {} },
      labels: { data: {} },
      packs: {
        data: {
          [packStub.id]: packStub,
        },
      },
      scheduled_queries: {},
    },
  };
  const page = mount(connectedComponent(ConnectedEditPackPage, {
    props: { params: { id: String(packStub.id) }, route: {} },
    mockStore: reduxMockStore(store),
  }));

  it('renders', () => {
    expect(page.length).toEqual(1);
  });

  it('renders a EditPackFormWrapper component', () => {
    expect(page.find('EditPackFormWrapper').length).toEqual(1);
  });

  it('renders a ScheduleQuerySidePanel component', () => {
    expect(page.find('ScheduleQuerySidePanel').length).toEqual(1);
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
