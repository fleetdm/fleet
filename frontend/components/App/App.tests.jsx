import { mount } from "enzyme";

import ConnectedApp from "./App";
import * as authActions from "../../redux/nodes/auth/actions";
import helpers from "../../test/helpers";
import local from "../../utilities/local";

const { connectedComponent, reduxMockStore } = helpers;

describe("App - component", () => {
  const store = { app: {}, auth: {}, notifications: {} };
  const mockStore = reduxMockStore(store);
  const component = mount(connectedComponent(ConnectedApp, { mockStore }));

  afterEach(() => {
    local.setItem("auth_token", null);
  });

  it("renders", () => {
    expect(component).toBeTruthy();
  });

  it("loads the current user if there is an auth token but no user", () => {
    local.setItem("auth_token", "ABC123");

    const spy = jest
      .spyOn(authActions, "fetchCurrentUser")
      .mockImplementation(() => {
        return (dispatch) => {
          dispatch({ type: "LOAD_USER_ACTION" });
          return Promise.resolve();
        };
      });
    const application = connectedComponent(ConnectedApp, { mockStore });

    mount(application);
    expect(spy).toHaveBeenCalled();
  });

  it("does not load the current user if is it already loaded", () => {
    local.setItem("auth_token", "ABC123");

    const spy = jest
      .spyOn(authActions, "fetchCurrentUser")
      .mockImplementation(() => {
        return { type: "LOAD_USER_ACTION" };
      });
    const storeWithUser = {
      app: {},
      auth: {
        user: {
          id: 1,
          email: "hi@thegnar.co",
        },
      },
      notifications: {},
    };
    const mockStoreWithUser = reduxMockStore(storeWithUser);
    const application = connectedComponent(ConnectedApp, {
      mockStore: mockStoreWithUser,
    });

    mount(application);
    expect(spy).not.toHaveBeenCalled();
  });

  it("does not load the current user if there is no auth token", () => {
    local.clear();

    const spy = jest
      .spyOn(authActions, "fetchCurrentUser")
      .mockImplementation(() => {
        throw new Error("should not have been called");
      });
    const application = connectedComponent(ConnectedApp, { mockStore });

    mount(application);
    expect(spy).not.toHaveBeenCalled();
  });
});
