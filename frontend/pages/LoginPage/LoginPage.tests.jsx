import expect from 'expect';
import { mount } from 'enzyme';
import { connectedComponent, reduxMockStore } from '../../test/helpers';
import local from '../../utilities/local';
import LoginPage from './LoginPage';

describe('LoginPage - component', () => {
  context('when the user is not logged in', () => {
    const mockStore = reduxMockStore({ auth: {} });

    it('renders the LoginForm', () => {
      const page = mount(connectedComponent(LoginPage, { mockStore }));

      expect(page.find('LoginForm').length).toEqual(1);
    });
  });

  context('when the users session is not recognized', () => {
    const mockStore = reduxMockStore({
      auth: {
        error: 'Unable to authenticate the current user',
      },
    });

    it('renders the LoginForm without displaying errors', () => {
      const page = mount(connectedComponent(LoginPage, { mockStore }));
      const loginForm = page.find('LoginForm');

      expect(loginForm.length).toEqual(1);
      expect(loginForm.prop('serverErrors')).toEqual({});
    });
  });

  context('when the user is logged in', () => {
    beforeEach(() => {
      local.setItem('auth_token', 'fake-auth-token');
    });

    const user = { id: 1, firstName: 'Bill', lastName: 'Shakespeare' };

    it('redirects to the home page', () => {
      const mockStore = reduxMockStore({ auth: { user } });
      const props = { pathname: '/login' };
      const redirectAction = {
        type: '@@router/CALL_HISTORY_METHOD',
        payload: {
          method: 'push',
          args: ['/'],
        },
      };

      mount(connectedComponent(LoginPage, { props, mockStore }));
      expect(mockStore.getActions()).toInclude(redirectAction);
    });
  });
});
