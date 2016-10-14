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
});
