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
const store = {
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

  it('gets all users if there are no users in state', () => {
    const mockStore = reduxMockStore({
      ...store,
      entities: {
        ...store.entities,
        users: {
          ...store.entities.users,
          data: {},
        },
      },
    });

    mount(connectedComponent(UserManagementPage, { mockStore }));

    expect(mockStore.getActions()).toInclude(loadUsersAction);
  });

  it('gets all users if the only user in state is the current user', () => {
    const mockStore = reduxMockStore(store);

    mount(connectedComponent(UserManagementPage, { mockStore }));

    expect(mockStore.getActions()).toInclude(loadUsersAction);
  });

  it('does not get users if users are already loaded', () => {
    const mockStore = reduxMockStore({
      ...store,
      entities: {
        ...store.entities,
        users: {
          ...store.entities.users,
          data: {
            1: { ...currentUser },
            2: { id: 2, email: 'another@gnar.dog', full_name: 'GnarDog' },
          },
        },
      },
    });

    mount(connectedComponent(UserManagementPage, { mockStore }));

    expect(mockStore.getActions()).toNotInclude(loadUsersAction);
  });
});
