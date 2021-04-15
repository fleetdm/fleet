import React from "react";
import { mount } from "enzyme";
import { Provider } from "react-redux";

import AuthenticatedRoutes from "./index";
import helpers from "../../test/helpers";

describe("AuthenticatedRoutes - component", () => {
  const redirectToLoginAction = {
    type: "@@router/CALL_HISTORY_METHOD",
    payload: {
      method: "push",
      args: ["/login"],
    },
  };
  const redirectToPasswordResetAction = {
    type: "@@router/CALL_HISTORY_METHOD",
    payload: {
      method: "push",
      args: ["/login/reset"],
    },
  };
  const renderedText = "This text was rendered";
  const storeWithUser = {
    auth: {
      loading: false,
      user: {
        id: 1,
        email: "hi@thegnar.co",
        force_password_reset: false,
      },
    },
    routing: {
      locationBeforeTransitions: {},
    },
  };
  const storeWithUserRequiringPwReset = {
    auth: {
      loading: false,
      user: {
        id: 1,
        email: "hi@thegnar.co",
        force_password_reset: true,
      },
    },
    routing: {
      locationBeforeTransitions: {},
    },
  };
  const storeWithoutUser = {
    auth: {
      loading: false,
      user: null,
    },
    routing: {
      locationBeforeTransitions: {},
    },
  };
  const storeLoadingUser = {
    auth: {
      loading: true,
      user: null,
    },
    routing: {
      locationBeforeTransitions: {},
    },
  };

  it("renders if there is a user in state", () => {
    const { reduxMockStore } = helpers;
    const mockStore = reduxMockStore(storeWithUser);
    const component = mount(
      <Provider store={mockStore}>
        <AuthenticatedRoutes>
          <div>{renderedText}</div>
        </AuthenticatedRoutes>
      </Provider>
    );

    expect(component.text()).toEqual(renderedText);
  });

  it("redirects to reset password is force_password_reset is true", () => {
    const { reduxMockStore } = helpers;
    const mockStore = reduxMockStore(storeWithUserRequiringPwReset);
    mount(
      <Provider store={mockStore}>
        <AuthenticatedRoutes>
          <div>{renderedText}</div>
        </AuthenticatedRoutes>
      </Provider>
    );

    expect(mockStore.getActions()).toContainEqual(
      redirectToPasswordResetAction
    );
  });

  it("redirects to login without a user", () => {
    const { reduxMockStore } = helpers;
    const mockStore = reduxMockStore(storeWithoutUser);
    const component = mount(
      <Provider store={mockStore}>
        <AuthenticatedRoutes>
          <div>{renderedText}</div>
        </AuthenticatedRoutes>
      </Provider>
    );

    expect(mockStore.getActions()).toContainEqual(redirectToLoginAction);
    expect(component.html()).toBeFalsy();
  });

  it("does not redirect to login if the user is loading", () => {
    const { reduxMockStore } = helpers;
    const mockStore = reduxMockStore(storeLoadingUser);
    const component = mount(
      <Provider store={mockStore}>
        <AuthenticatedRoutes>
          <div>{renderedText}</div>
        </AuthenticatedRoutes>
      </Provider>
    );

    expect(mockStore.getActions()).not.toContainEqual(redirectToLoginAction);
    expect(component.html()).toBeFalsy();
  });
});
