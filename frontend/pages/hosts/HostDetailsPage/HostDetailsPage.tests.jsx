import React from 'react';
import { mount } from 'enzyme';
import { noop } from 'lodash';
import { hostStub } from 'test/stubs';
import hostActions from 'redux/nodes/entities/hosts/actions';
import { connectedComponent, reduxMockStore } from 'test/helpers';
import ConnectedHostDetailsPage, { HostDetailsPage } from './HostDetailsPage';

const mockStoreWithHost = reduxMockStore({
  entities: {
    hosts: {
      data: {
        [hostStub.id]: hostStub,
      },
    },
  },
});

const mockStoreNoHost = reduxMockStore({
  entities: {
    hosts: {
      data: {},
    },
  },
});

describe('HostDetailsPage - component', () => {
  const propsWithHost = {
    params: {
      host_id: hostStub.id,
    },
  };

  const propsNoHost = {
    params: {
      host_id: hostStub.id,
    },
  };

  describe('loading host data', () => {
    it('does not load host data exists in the store', () => {
      const spy = jest.spyOn(hostActions, 'load')
        .mockImplementation(() => () => Promise.resolve([]));
      const component = connectedComponent(ConnectedHostDetailsPage, {
        props: propsWithHost,
        mockStore: mockStoreWithHost,
      });
      mount(component);
      expect(spy).not.toHaveBeenCalled();
    });

    it('does load host data if host data does not exist in the store', () => {
      const spy = jest.spyOn(hostActions, 'load')
        .mockImplementation(() => () => Promise.resolve([]));
      const component = connectedComponent(ConnectedHostDetailsPage, {
        props: propsNoHost,
        mockStore: mockStoreNoHost,
      });
      mount(component);
      expect(spy).toHaveBeenCalled();
    });
  });
});
