import expect from 'expect';
import { mount } from 'enzyme';

import ConnectedPage from 'pages/UserSettingsPage';
import testHelpers from 'test/helpers';
import { userStub } from 'test/stubs';

const { connectedComponent, reduxMockStore } = testHelpers;

describe('UserSettingsPage - component', () => {
  it('renders a UserSettingsForm component', () => {
    const store = { auth: { user: userStub } };
    const mockStore = reduxMockStore(store);

    const page = mount(connectedComponent(ConnectedPage, { mockStore }));

    expect(page.find('UserSettingsForm').length).toEqual(1);
  });
});
