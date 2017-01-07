import React from 'react';
import expect from 'expect';
import { mount } from 'enzyme';
import { noop } from 'lodash';

import ConnectedPage, { UserSettingsPage } from 'pages/UserSettingsPage/UserSettingsPage';
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

  it('renders a UserSettingsForm component', () => {
    const admin = { ...userStub, admin: true };
    const pageWithUser = mount(<UserSettingsPage dispatch={noop} user={userStub} />);
    const pageWithAdmin = mount(<UserSettingsPage dispatch={noop} user={admin} />);

    expect(pageWithUser.text()).toInclude('Role - USER');
    expect(pageWithUser.text()).toNotInclude('Role - ADMIN');
    expect(pageWithAdmin.text()).toNotInclude('Role - USER');
    expect(pageWithAdmin.text()).toInclude('Role - ADMIN');
  });
});
