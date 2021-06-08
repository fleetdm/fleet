import React from "react";
import { mount } from "enzyme";

import ConnectedPage, { ResetPasswordPage } from "./ResetPasswordPage";
import testHelpers from "../../test/helpers";

describe("ResetPasswordPage - component", () => {
  it("renders a ResetPasswordForm", () => {
    const page = mount(<ResetPasswordPage token="ABC123" />);

    expect(page.find("ResetPasswordForm").length).toEqual(1);
  });

  it("Redirects to the login page when there is no token or user", () => {
    const { connectedComponent, reduxMockStore } = testHelpers;
    const redirectToLoginAction = {
      type: "@@router/CALL_HISTORY_METHOD",
      payload: {
        method: "push",
        args: ["/login"],
      },
    };
    const store = {
      auth: {},
      components: {
        ResetPasswordPage: {
          loading: false,
          error: null,
        },
      },
    };
    const mockStore = reduxMockStore(store);

    mount(connectedComponent(ConnectedPage, { mockStore }));

    const dispatchedActions = mockStore.getActions();

    expect(dispatchedActions).toContainEqual(redirectToLoginAction);
  });
});
