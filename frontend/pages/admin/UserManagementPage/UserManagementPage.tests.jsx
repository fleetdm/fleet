import React from 'react';
import { mount } from 'enzyme';

import { connectedComponent, reduxMockStore } from 'test/helpers';
import ConnectedUserManagementPage from 'pages/admin/UserManagementPage/UserManagementPage';
import inviteActions from 'redux/nodes/entities/invites/actions';
import userActions from 'redux/nodes/entities/users/actions';

const currentUser = {
  admin: true,
  email: 'hi@gnar.dog',
  enabled: true,
  name: 'Gnar Dog',
  position: 'Head of Gnar',
  username: 'gnardog',
  teams: [],
  global_role: 'admin',
};
const store = {
  app: {
    config: {
      configured: true,
    },
  },
  auth: {
    user: {
      ...currentUser,
    },
  },
  entities: {
    users: {
      loading: false,
      data: {
        1: {
          ...currentUser,
        },
      },
      originalOrder: [1],
    },
    invites: {
      loading: false,
      data: {
        1: {
          admin: false,
          email: 'other@user.org',
          name: 'Other user',
        },
      },
      originalOrder: [1],
    },
  },
};

describe('UserManagementPage - component', () => {
  beforeEach(() => {
    jest.spyOn(userActions, 'loadAll')
      .mockImplementation(() => () => Promise.resolve([]));

    jest.spyOn(inviteActions, 'loadAll')
      .mockImplementation(() => () => Promise.resolve([]));
  });

  describe('rendering', () => {
    it(
      'displays a disabled "Create user" button if email is not configured',
      () => {
        const notConfiguredStore = { ...store, app: { config: { configured: false } } };
        const notConfiguredMockStore = reduxMockStore(notConfiguredStore);
        const notConfiguredPage = mount(connectedComponent(ConnectedUserManagementPage, {
          mockStore: notConfiguredMockStore,
        }));

        const configuredStore = store;
        const configuredMockStore = reduxMockStore(configuredStore);
        const configuredPage = mount(connectedComponent(ConnectedUserManagementPage, {
          mockStore: configuredMockStore,
        }));

        expect(notConfiguredPage.find('Button').at(1).prop('disabled')).toEqual(true);
        expect(configuredPage.find('Button').first().prop('disabled')).toEqual(false);
      },
    );

    it('displays a SmtpWarning if email is not configured', () => {
      const notConfiguredStore = { ...store, app: { config: { configured: false } } };
      const notConfiguredMockStore = reduxMockStore(notConfiguredStore);
      const notConfiguredPage = mount(connectedComponent(ConnectedUserManagementPage, {
        mockStore: notConfiguredMockStore,
      }));

      const configuredStore = store;
      const configuredMockStore = reduxMockStore(configuredStore);
      const configuredPage = mount(connectedComponent(ConnectedUserManagementPage, {
        mockStore: configuredMockStore,
      }));

      expect(notConfiguredPage.find('WarningBanner').html()).toBeTruthy();
      expect(configuredPage.find('WarningBanner').html()).toBeFalsy();
    });
  });

  it(
    'goes to the app settings page for the user to resolve their smtp settings',
    () => {
      const notConfiguredStore = { ...store, app: { config: { configured: false } } };
      const mockStore = reduxMockStore(notConfiguredStore);
      const page = mount(connectedComponent(ConnectedUserManagementPage, { mockStore }));

      const smtpWarning = page.find('WarningBanner');

      smtpWarning.find('Button').simulate('click');

      const goToAppSettingsAction = {
        type: '@@router/CALL_HISTORY_METHOD',
        payload: { method: 'push', args: ['/settings/organization'] },
      };

      expect(mockStore.getActions()).toContainEqual(goToAppSettingsAction);
    },
  );
});
