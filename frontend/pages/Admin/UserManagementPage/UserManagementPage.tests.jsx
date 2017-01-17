import expect from 'expect';
import { mount } from 'enzyme';

import { connectedComponent, reduxMockStore } from '../../../test/helpers';
import UserManagementPage from './UserManagementPage';

const currentUser = {
  admin: true,
  email: 'hi@gnar.dog',
  enabled: true,
  name: 'Gnar Dog',
  position: 'Head of Gnar',
  username: 'gnardog',
};
const loadUsersAction = { type: 'users_LOAD_REQUEST' };
const loadInvitesAction = { type: 'invites_LOAD_REQUEST' };
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
    },
  },
};

describe('UserManagementPage - component', () => {
  it('displays a disabled "Add User" button if email is not configured', () => {
    const notConfiguredStore = { ...store, app: { config: { configured: false } } };
    const notConfiguredMockStore = reduxMockStore(notConfiguredStore);
    const notConfiguredPage = mount(connectedComponent(UserManagementPage, {
      mockStore: notConfiguredMockStore,
    }));

    const configuredStore = store;
    const configuredMockStore = reduxMockStore(configuredStore);
    const configuredPage = mount(connectedComponent(UserManagementPage, {
      mockStore: configuredMockStore,
    }));

    expect(notConfiguredPage.find('Button').first().prop('disabled')).toEqual(true);
    expect(configuredPage.find('Button').first().prop('disabled')).toEqual(false);
  });

  it('displays a SmtpWarning if email is not configured', () => {
    const notConfiguredStore = { ...store, app: { config: { configured: false } } };
    const notConfiguredMockStore = reduxMockStore(notConfiguredStore);
    const notConfiguredPage = mount(connectedComponent(UserManagementPage, {
      mockStore: notConfiguredMockStore,
    }));

    const configuredStore = store;
    const configuredMockStore = reduxMockStore(configuredStore);
    const configuredPage = mount(connectedComponent(UserManagementPage, {
      mockStore: configuredMockStore,
    }));

    expect(notConfiguredPage.find('SmtpWarning').html()).toExist();
    expect(configuredPage.find('SmtpWarning').html()).toNotExist();
  });

  it('goes to the app settings page for the user to resolve their smtp settings', () => {
    const notConfiguredStore = { ...store, app: { config: { configured: false } } };
    const mockStore = reduxMockStore(notConfiguredStore);
    const page = mount(connectedComponent(UserManagementPage, { mockStore }));

    const smtpWarning = page.find('SmtpWarning');

    smtpWarning.find('Button').simulate('click');

    const goToAppSettingsAction = {
      type: '@@router/CALL_HISTORY_METHOD',
      payload: { method: 'push', args: ['/admin/settings'] },
    };

    expect(mockStore.getActions()).toInclude(goToAppSettingsAction);
  });

  it('renders user blocks for users and invites', () => {
    const mockStore = reduxMockStore(store);
    const page = mount(connectedComponent(UserManagementPage, { mockStore }));

    expect(page.find('UserBlock').length).toEqual(2);
  });

  it('displays a count of the number of users & invites', () => {
    const mockStore = reduxMockStore(store);
    const page = mount(connectedComponent(UserManagementPage, { mockStore }));

    expect(page.text()).toInclude('Listing 2 users');
  });

  it('gets users on mount', () => {
    const mockStore = reduxMockStore(store);

    mount(connectedComponent(UserManagementPage, { mockStore }));

    expect(mockStore.getActions()).toInclude(loadUsersAction);
  });

  it('gets invites on mount', () => {
    const mockStore = reduxMockStore(store);

    mount(connectedComponent(UserManagementPage, { mockStore }));

    expect(mockStore.getActions()).toInclude(loadInvitesAction);
  });
});
