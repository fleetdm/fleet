import expect, { spyOn, restoreSpies } from 'expect';
import { mount } from 'enzyme';

import helpers from 'test/helpers';
import Kolide from 'kolide';
import LicensePage from 'pages/LicensePage';
import stubs from 'test/stubs';

const {
  connectedComponent,
  reduxMockStore,
} = helpers;
const {
  licenseStub,
  userStub,
} = stubs;

describe('LicensePage - component', () => {
  describe('rendering', () => {
    it('renders the license success content when a license is present', () => {
      const store = {
        auth: {
          license: licenseStub(),
          loading: false,
          user: null,
        },
      };
      const Component = connectedComponent(LicensePage, {
        mockStore: reduxMockStore(store),
      });

      expect(mount(Component).find('LicenseForm').length).toEqual(0, 'Expected the LicenseForm to not be on the page when a license is present');
      expect(mount(Component).find('LicenseSuccess').length).toEqual(1, 'Expected the LicenseSuccess component to be on the page when a license is present');
    });

    it('renders when not authenticated', () => {
      const store = {
        auth: {
          license: {},
          loading: false,
          user: null,
        },
      };
      const Component = connectedComponent(LicensePage, {
        mockStore: reduxMockStore(store),
      });

      expect(mount(Component).find('LicensePage').length).toEqual(1);
    });

    it('does not render when a user is logged in', () => {
      const store = {
        auth: {
          license: {},
          loading: false,
          user: userStub,
        },
      };
      const Component = connectedComponent(LicensePage, {
        mockStore: reduxMockStore(store),
      });

      expect(mount(Component).find('LicensePage').length).toEqual(0);
    });

    it('does not render when loading the user', () => {
      const store = {
        auth: {
          license: {},
          loading: true,
          user: null,
        },
      };
      const Component = connectedComponent(LicensePage, {
        mockStore: reduxMockStore(store),
      });

      expect(mount(Component).find('LicensePage').length).toEqual(0);
    });

    it('renders a LicenseForm when a license is not present', () => {
      const store = {
        auth: {
          license: {},
          loading: false,
          user: null,
        },
      };
      const Component = connectedComponent(LicensePage, {
        mockStore: reduxMockStore(store),
      });

      expect(mount(Component).find('LicenseForm').length).toEqual(1, 'Expected the LicenseForm to be on the page');
    });
  });

  describe('submitting the form', () => {
    afterEach(restoreSpies);

    it('calls the Kolide setup license endpoint', () => {
      spyOn(Kolide.license, 'setup').andReturn(Promise.resolve());

      const store = {
        auth: {
          license: {},
          loading: false,
          user: null,
        },
      };
      const Component = connectedComponent(LicensePage, {
        mockStore: reduxMockStore(store),
      });
      const Form = mount(Component).find('LicenseForm');
      const LicenseField = Form.find({ name: 'license' }).find('textarea');
      const jwtToken = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWV9.TJVA95OrM7E2cBab30RMHrHDcEfxjoYZgeFONFh7HgQ';

      helpers.fillInFormInput(LicenseField, jwtToken);
      Form.simulate('submit');

      expect(Kolide.license.setup).toHaveBeenCalledWith(jwtToken);
    });
  });
});
