import React from "react";
import { mount } from "enzyme";
import { noop } from "lodash";

import * as authActions from "redux/nodes/auth/actions";
import { connectedComponent, reduxMockStore } from "test/helpers";
import ConnectedUserManagementPage, {
  UserManagementPage,
} from "pages/admin/UserManagementPage/UserManagementPage";
import inviteActions from "redux/nodes/entities/invites/actions";
import userActions from "redux/nodes/entities/users/actions";

const currentUser = {
  admin: true,
  email: "hi@gnar.dog",
  enabled: true,
  name: "Gnar Dog",
  position: "Head of Gnar",
  username: "gnardog",
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
          email: "other@user.org",
          name: "Other user",
        },
      },
    },
  },
};

describe("UserManagementPage - component", () => {
  beforeEach(() => {
    jest
      .spyOn(userActions, "loadAll")
      .mockImplementation(() => () => Promise.resolve([]));

    jest
      .spyOn(inviteActions, "loadAll")
      .mockImplementation(() => () => Promise.resolve([]));
  });

  describe("rendering", () => {
    it("does not render if invites are loading", () => {
      const props = {
        dispatch: noop,
        config: {},
        currentUser,
        invites: [],
        loadingInvites: true,
        loadingUsers: false,
        users: [currentUser],
      };
      const page = mount(<UserManagementPage {...props} />);

      expect(page.html()).toBeFalsy();
    });

    it("does not render if users are loading", () => {
      const props = {
        dispatch: noop,
        config: {},
        currentUser,
        invites: [],
        loadingInvites: false,
        loadingUsers: true,
        users: [currentUser],
      };
      const page = mount(<UserManagementPage {...props} />);

      expect(page.html()).toBeFalsy();
    });

    it("renders user blocks for users and invites", () => {
      const mockStore = reduxMockStore(store);
      const page = mount(
        connectedComponent(ConnectedUserManagementPage, { mockStore })
      );

      expect(page.find("UserRow").length).toEqual(2);
    });

    it("displays a count of the number of users & invites", () => {
      const mockStore = reduxMockStore(store);
      const page = mount(
        connectedComponent(ConnectedUserManagementPage, { mockStore })
      );
      const count = page.find(".user-management__user-count");
      expect(count.text()).toContain("2 users");
    });

    it('displays a disabled "Invite user" button if email is not configured', () => {
      const notConfiguredStore = {
        ...store,
        app: { config: { configured: false } },
      };
      const notConfiguredMockStore = reduxMockStore(notConfiguredStore);
      const notConfiguredPage = mount(
        connectedComponent(ConnectedUserManagementPage, {
          mockStore: notConfiguredMockStore,
        })
      );

      const configuredStore = store;
      const configuredMockStore = reduxMockStore(configuredStore);
      const configuredPage = mount(
        connectedComponent(ConnectedUserManagementPage, {
          mockStore: configuredMockStore,
        })
      );

      expect(notConfiguredPage.find("Button").at(1).prop("disabled")).toEqual(
        true
      );
      expect(configuredPage.find("Button").first().prop("disabled")).toEqual(
        false
      );
    });

    it("displays a SmtpWarning if email is not configured", () => {
      const notConfiguredStore = {
        ...store,
        app: { config: { configured: false } },
      };
      const notConfiguredMockStore = reduxMockStore(notConfiguredStore);
      const notConfiguredPage = mount(
        connectedComponent(ConnectedUserManagementPage, {
          mockStore: notConfiguredMockStore,
        })
      );

      const configuredStore = store;
      const configuredMockStore = reduxMockStore(configuredStore);
      const configuredPage = mount(
        connectedComponent(ConnectedUserManagementPage, {
          mockStore: configuredMockStore,
        })
      );

      expect(notConfiguredPage.find("WarningBanner").html()).toBeTruthy();
      expect(configuredPage.find("WarningBanner").html()).toBeFalsy();
    });
  });

  it("goes to the app settings page for the user to resolve their smtp settings", () => {
    const notConfiguredStore = {
      ...store,
      app: { config: { configured: false } },
    };
    const mockStore = reduxMockStore(notConfiguredStore);
    const page = mount(
      connectedComponent(ConnectedUserManagementPage, { mockStore })
    );

    const smtpWarning = page.find("WarningBanner");

    smtpWarning.find("Button").simulate("click");

    const goToAppSettingsAction = {
      type: "@@router/CALL_HISTORY_METHOD",
      payload: { method: "push", args: ["/settings/organization"] },
    };

    expect(mockStore.getActions()).toContainEqual(goToAppSettingsAction);
  });

  it("gets users on mount", () => {
    const mockStore = reduxMockStore(store);

    mount(connectedComponent(ConnectedUserManagementPage, { mockStore }));

    expect(userActions.loadAll).toHaveBeenCalled();
  });

  it("gets invites on mount", () => {
    const mockStore = reduxMockStore(store);

    mount(connectedComponent(ConnectedUserManagementPage, { mockStore }));

    expect(inviteActions.loadAll).toHaveBeenCalled();
  });

  describe("updating a user", () => {
    const dispatch = () => Promise.resolve();
    const props = {
      dispatch,
      config: {},
      currentUser,
      invites: [],
      users: [currentUser],
    };
    const pageNode = mount(<UserManagementPage {...props} />).instance();
    const updatedAttrs = { name: "Updated Name" };

    it("updates the current user with only the updated attributes", () => {
      jest.spyOn(authActions, "updateUser");

      const updatedUser = { ...currentUser, ...updatedAttrs };

      pageNode.onEditUser(currentUser, updatedUser);

      expect(authActions.updateUser).toHaveBeenCalledWith(
        currentUser,
        updatedAttrs
      );
    });

    it("updates a different user with only the updated attributes", () => {
      jest.spyOn(userActions, "silentUpdate");

      const otherUser = { ...currentUser, id: currentUser.id + 1 };
      const updatedUser = { ...otherUser, ...updatedAttrs };

      pageNode.onEditUser(otherUser, updatedUser);

      expect(userActions.silentUpdate).toHaveBeenCalledWith(
        otherUser,
        updatedAttrs
      );
    });
  });
});
