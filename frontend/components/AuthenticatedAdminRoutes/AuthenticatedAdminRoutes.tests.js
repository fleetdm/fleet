import { mount } from "enzyme";

import ConnectedAdminRoutes from "./AuthenticatedAdminRoutes";
import { connectedComponent, reduxMockStore } from "../../test/helpers";

describe("AuthenticatedAdminRoutes - layout", () => {
  const redirectToHomeAction = {
    type: "@@router/CALL_HISTORY_METHOD",
    payload: {
      method: "push",
      args: ["/"],
    },
  };

  it("redirects to the homepage if the user is not an admin", () => {
    const user = { id: 1, admin: false };
    const storeWithoutAdminUser = { auth: { user } };
    const mockStore = reduxMockStore(storeWithoutAdminUser);
    mount(connectedComponent(ConnectedAdminRoutes, { mockStore }));

    expect(mockStore.getActions()).toContainEqual(redirectToHomeAction);
  });

  it("does not redirect if the user is an admin", () => {
    const user = { id: 1, admin: true };
    const storeWithAdminUser = { auth: { user } };
    const mockStore = reduxMockStore(storeWithAdminUser);
    mount(connectedComponent(ConnectedAdminRoutes, { mockStore }));

    expect(mockStore.getActions()).not.toContainEqual(redirectToHomeAction);
  });
});
