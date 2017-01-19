import React from 'react';
import expect, { spyOn, restoreSpies } from 'expect';
import { mount } from 'enzyme';
import { noop } from 'lodash';

import ConnectedPage, { UserSettingsPage } from 'pages/UserSettingsPage/UserSettingsPage';
import testHelpers from 'test/helpers';
import { userStub } from 'test/stubs';
import * as authActions from 'redux/nodes/auth/actions';

const { connectedComponent, reduxMockStore } = testHelpers;

describe('UserSettingsPage - component', () => {
  afterEach(restoreSpies);

  it('renders a UserSettingsForm component', () => {
    const store = { auth: { user: userStub }, entities: { users: {} } };
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

  it('updates a user with only the updated attributes', () => {
    spyOn(authActions, 'updateUser');

    const dispatch = () => Promise.resolve();
    const props = { dispatch, user: userStub };
    const pageNode = mount(<UserSettingsPage {...props} />).node;
    const updatedAttrs = { name: 'Updated Name' };
    const updatedUser = { ...userStub, ...updatedAttrs };

    pageNode.handleSubmit(updatedUser);

    expect(authActions.updateUser).toHaveBeenCalledWith(userStub, updatedAttrs);
  });
});
