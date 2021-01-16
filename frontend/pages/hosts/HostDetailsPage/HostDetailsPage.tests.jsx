import React from 'react';
import { mount } from 'enzyme';
import { noop } from 'lodash';
import { hostStub } from 'test/stubs';
import hostActions from 'redux/nodes/entities/hosts/actions';
import { connectedComponent, reduxMockStore } from 'test/helpers';
import ConnectedHostDetailsPage, { HostDetailsPage } from './HostDetailsPage';

const offlineHost = { ...hostStub, id: 111, status: 'offline' };

const mockStoreWithHost = reduxMockStore({
  entities: {
    hosts: {
      data: {
        [hostStub.id]: hostStub,
      },
    },
    loading: false,
  },
});

const mockStoreNoHost = reduxMockStore({
  entities: {
    hosts: {
      data: {},
    },
    loading: false,
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

  describe('Loading host data', () => {
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

  describe('Delete a host', () => {
    it('Deleted host after confirmation modal', () => {
      const component = connectedComponent(ConnectedHostDetailsPage, {
        props: propsWithHost,
        mockStore: mockStoreWithHost,
      });
      const page = mount(component);
      const deleteBtn = page.find('Button').at(1);

      jest.spyOn(hostActions, 'destroy').mockImplementation(() => (dispatch) => {
        dispatch({ type: 'hosts_LOAD_REQUEST' });

        return Promise.resolve();
      });

      expect(page.find('Modal').length).toEqual(0);

      deleteBtn.simulate('click');

      const confirmModal = page.find('Modal');

      expect(confirmModal.length).toEqual(1);

      const confirmBtn = confirmModal.find('.button--alert');
      confirmBtn.simulate('click');

      expect(hostActions.destroy).toHaveBeenCalledWith(hostStub);
    });
  });

  // describe('Query a host', () => {
  //   it(
  //     'calls the onDestroyHost prop when the action button is clicked on an offline host',
  //     () => {
  //       const destroySpy = jest.fn();
  //       const querySpy = jest.fn();
  //       const offlineHost = { ...hostStub, status: 'offline' };

  //       const offlineComponent = mount(
  //         <HostsTable
  //           hosts={[offlineHost]}
  //           onDestroyHost={destroySpy}
  //           onQueryHost={querySpy}
  //         />,
  //       );
  //       const btn = offlineComponent.find('Button');

  //       expect(btn.find('KolideIcon').prop('name')).toEqual('trash');

  //       btn.simulate('click');

  //       expect(destroySpy).toHaveBeenCalled();
  //       expect(querySpy).not.toHaveBeenCalled();
  //     },
  //   );

  //   it(
  //     'calls the onDestroyHost prop when the action button is clicked on a mia host',
  //     () => {
  //       const destroySpy = jest.fn();
  //       const querySpy = jest.fn();
  //       const miaHost = { ...hostStub, status: 'mia' };

  //       const miaComponent = mount(
  //         <HostsTable
  //           hosts={[miaHost]}
  //           onDestroyHost={destroySpy}
  //           onQueryHost={querySpy}
  //         />,
  //       );
  //       const btn = miaComponent.find('Button');

  //       expect(btn.find('KolideIcon').prop('name')).toEqual('trash');

  //       btn.simulate('click');

  //       expect(destroySpy).toHaveBeenCalled();
  //       expect(querySpy).not.toHaveBeenCalled();
  //     },
  //   );

  //   it(
  //     'calls the onQueryHost prop when the action button is clicked on an online host',
  //     () => {
  //       const destroySpy = jest.fn();
  //       const querySpy = jest.fn();
  //       const onlineHost = { ...hostStub, status: 'online' };

  //       const onlineComponent = mount(
  //         <HostsTable
  //           hosts={[onlineHost]}
  //           onDestroyHost={destroySpy}
  //           onQueryHost={querySpy}
  //         />,
  //       );
  //       const btn = onlineComponent.find('Button');

  //       expect(btn.find('KolideIcon').prop('name')).toEqual('query');

  //       btn.simulate('click');

  //       expect(destroySpy).not.toHaveBeenCalled();
  //       expect(querySpy).toHaveBeenCalled();
  //     },
  //   );
  // });
});
