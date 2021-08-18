import React from "react";
import { mount } from "enzyme";
import { noop } from "lodash";

import ConnectedPage, {
  UserSettingsPage,
} from "pages/UserSettingsPage/UserSettingsPage";
import testHelpers from "test/helpers";
import { userStub, configStub, adminUserStub } from "test/stubs";
import * as authActions from "redux/nodes/auth/actions";

const { connectedComponent, fillInFormInput, reduxMockStore } = testHelpers;

describe("UserSettingsPage - component", () => {
  const store = {
    auth: { user: userStub },
    app: { config: configStub },
    entities: { users: {} },
    version: { data: {} },
  };
  const mockStore = reduxMockStore(store);

  it("renders a UserSettingsForm component", () => {
    const Page = mount(connectedComponent(ConnectedPage, { mockStore }));

    expect(Page.find("UserSettingsForm").length).toEqual(1);
  });

  it("contains expected text", () => {
    const pageWithUser = mount(
      <UserSettingsPage dispatch={noop} user={userStub} config={configStub} />
    );
    const pageWithAdmin = mount(
      <UserSettingsPage
        dispatch={noop}
        user={adminUserStub}
        config={configStub}
      />
    );

    expect(pageWithUser.find(".user-settings__role").text()).toContain(
      "Observer"
    );
    expect(pageWithAdmin.find(".user-settings__role").text()).toContain(
      "Admin"
    );
  });

  it("updates a user with only the updated attributes", () => {
    jest.spyOn(authActions, "updateUser");

    const dispatch = () => Promise.resolve();
    const props = { dispatch, user: userStub, config: configStub };
    const pageNode = mount(<UserSettingsPage {...props} />).instance();
    const updatedAttrs = { name: "Updated Name" };
    const updatedUser = { ...userStub, ...updatedAttrs };

    pageNode.handleSubmit(updatedUser);

    expect(authActions.updateUser).toHaveBeenCalledWith(userStub, updatedAttrs);
  });

  describe("changing email address", () => {
    it("renders the ChangeEmailForm when the user changes their email", () => {
      const Page = mount(connectedComponent(ConnectedPage, { mockStore }));
      const UserSettingsForm = Page.find("UserSettingsForm");
      const emailInput = UserSettingsForm.find({ name: "email" });

      expect(Page.find("ChangeEmailForm").length).toEqual(
        0,
        "Expected the ChangeEmailForm to not render"
      );

      fillInFormInput(emailInput, "new@email.org");
      UserSettingsForm.simulate("submit");

      expect(Page.find("ChangeEmailForm").length).toEqual(
        1,
        "Expected the ChangeEmailForm to render"
      );
    });

    it("does not render the ChangeEmailForm when the user does not change their email", () => {
      const Page = mount(connectedComponent(ConnectedPage, { mockStore }));
      const UserSettingsForm = Page.find("UserSettingsForm");
      const emailInput = UserSettingsForm.find({ name: "email" });

      expect(Page.find("ChangeEmailForm").length).toEqual(
        0,
        "Expected the ChangeEmailForm to not render"
      );

      fillInFormInput(emailInput, userStub.email);
      UserSettingsForm.simulate("submit");

      expect(Page.find("ChangeEmailForm").length).toEqual(
        0,
        "Expected the ChangeEmailForm to not render"
      );
    });

    it("displays pending email text when the user is pending an email change", () => {
      const props = { dispatch: noop, user: userStub, config: configStub };
      const Page = mount(<UserSettingsPage {...props} />);
      const UserSettingsForm = () => Page.find("UserSettingsForm");
      const emailHint = () =>
        UserSettingsForm().find(".manage-user__email-hint");

      expect(emailHint().length).toEqual(
        0,
        "Expected the form to not render an email hint"
      );

      Page.setState({ pendingEmail: "new@email.org" });

      expect(emailHint().length).toEqual(
        1,
        "Expected the form to render an email hint"
      );
    });
  });
});
