import expect, { restoreSpies, spyOn } from 'expect';
import { mount } from 'enzyme';

import { connectedComponent, reduxMockStore } from 'test/helpers';
import OsqueryOptionsPage from 'pages/admin/OsqueryOptionsPage';
import { getOsqueryOptions } from 'redux/nodes/osquery/actions';

const osqueryOptionsActions = {
  getOsqueryOptions,
};

const currentUser = {
  admin: true,
  email: 'hi@gnar.dog',
  enabled: true,
  name: 'Gnar Dog',
  position: 'Head of Gnar',
  username: 'gnardog',
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
  osquery: {
    erros: {},
    loading: false,
    options: {},
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
  },
};

describe('Osquery Options Page - Component', () => {
  beforeEach(() => {
    spyOn(osqueryOptionsActions, 'getOsqueryOptions')
      .andReturn(() => Promise.resolve([]));
  });

  afterEach(restoreSpies);

  it('renders', () => {
    const mockStore = reduxMockStore(store);
    const page = mount(connectedComponent(OsqueryOptionsPage, { mockStore }));

    expect(page.find('OsqueryOptionsPage').length).toEqual(1);
  });

  it('gets osquery options on mount', () => {
    const mockStore = reduxMockStore(store);

    mount(connectedComponent(OsqueryOptionsPage, { mockStore }));

    expect(osqueryOptionsActions.getOsqueryOptions).toHaveBeenCalled();
  });
});
