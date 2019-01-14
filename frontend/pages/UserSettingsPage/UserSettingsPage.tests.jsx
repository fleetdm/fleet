import React from 'react';
import expect, { spyOn, restoreSpies } from 'expect';
import { mount } from 'enzyme';
import { noop } from 'lodash';

import ConnectedPage, { UserSettingsPage } from 'pages/UserSettingsPage/UserSettingsPage';
import testHelpers from 'test/helpers';
import { userStub } from 'test/stubs';
import * as authActions from 'redux/nodes/auth/actions';

const {
  connectedComponent,
  fillInFormInput,
  reduxMockStore,
} = testHelpers;

describe('UserSettingsPage - component', () => {
  afterEach(restoreSpies);

  const store = { auth: { user: userStub }, entities: { users: {} } };
  const mockStore = reduxMockStore(store);

  it('renders a UserSettingsForm component', () => {
    const Page = mount(connectedComponent(ConnectedPage, { mockStore }));

    expect(Page.find('UserSettingsForm').length).toEqual(1);
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
    const pageNode = mount(<UserSettingsPage {...props} />).instance();
    const updatedAttrs = { name: 'Updated Name' };
    const updatedUser = { ...userStub, ...updatedAttrs };

    pageNode.handleSubmit(updatedUser);

    expect(authActions.updateUser).toHaveBeenCalledWith(userStub, updatedAttrs);
  });

  describe('changing email address', () => {
    it('renders the ChangeEmailForm when the user changes their email', () => {
      const Page = mount(connectedComponent(ConnectedPage, { mockStore }));
      const UserSettingsForm = Page.find('UserSettingsForm');
      const emailInput = UserSettingsForm.find({ name: 'email' });

      expect(Page.find('ChangeEmailForm').length).toEqual(0, 'Expected the ChangeEmailForm to not render');

      fillInFormInput(emailInput, 'new@email.org');
      UserSettingsForm.simulate('submit');

      expect(Page.find('ChangeEmailForm').length).toEqual(1, 'Expected the ChangeEmailForm to render');
    });

    it('does not render the ChangeEmailForm when the user does not change their email', () => {
      const Page = mount(connectedComponent(ConnectedPage, { mockStore }));
      const UserSettingsForm = Page.find('UserSettingsForm');
      const emailInput = UserSettingsForm.find({ name: 'email' });

      expect(Page.find('ChangeEmailForm').length).toEqual(0, 'Expected the ChangeEmailForm to not render');

      fillInFormInput(emailInput, userStub.email);
      UserSettingsForm.simulate('submit');

      expect(Page.find('ChangeEmailForm').length).toEqual(0, 'Expected the ChangeEmailForm to not render');
    });

    it('displays pending email text when the user is pending an email change', () => {
      const props = { dispatch: noop, user: userStub };
      const Page = mount(<UserSettingsPage {...props} />);
      const UserSettingsForm = () => Page.find('UserSettingsForm');
      const emailHint = () => UserSettingsForm().find('.manage-user__email-hint');

      expect(emailHint().length).toEqual(0, 'Expected the form to not render an email hint');

      Page.setState({ pendingEmail: 'new@email.org' });

      expect(emailHint().length).toEqual(1, 'Expected the form to render an email hint');
    });
  });
});
