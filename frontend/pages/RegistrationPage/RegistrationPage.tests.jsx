import React from "react";
import { mount, shallow } from "enzyme";

import paths from "router/paths";
import { connectedComponent, reduxMockStore } from "test/helpers";
import ConnectedRegistrationPage, {
  RegistrationPage,
} from "pages/RegistrationPage/RegistrationPage";

const baseStore = {
  app: {},
  auth: {},
};
const user = {
  id: 1,
  name: "Gnar Dog",
  email: "hi@gnar.dog",
};

describe("RegistrationPage - component", () => {
  it("redirects to the home page when a user is logged in", () => {
    const storeWithUser = {
      ...baseStore,
      auth: {
        loading: false,
        user,
      },
    };
    const mockStore = reduxMockStore(storeWithUser);

    mount(connectedComponent(ConnectedRegistrationPage, { mockStore }));

    const dispatchedActions = mockStore.getActions();

    const redirectToHomeAction = {
      type: "@@router/CALL_HISTORY_METHOD",
      payload: {
        method: "push",
        args: [paths.HOME],
      },
    };

    expect(dispatchedActions).toContainEqual(redirectToHomeAction);
  });

  it("displays the Kolide background triangles", () => {
    const mockStore = reduxMockStore(baseStore);

    mount(connectedComponent(ConnectedRegistrationPage, { mockStore }));

    expect(mockStore.getActions()).toContainEqual({
      type: "SHOW_BACKGROUND_IMAGE",
    });
  });

  it("does not render the RegistrationForm if the user is loading", () => {
    const mockStore = reduxMockStore({
      app: {},
      auth: { loading: true },
    });
    const page = mount(
      connectedComponent(ConnectedRegistrationPage, { mockStore })
    );

    expect(page.find("RegistrationForm").length).toEqual(0);
  });

  it("renders the RegistrationForm when there is no user", () => {
    const mockStore = reduxMockStore(baseStore);
    const page = mount(
      connectedComponent(ConnectedRegistrationPage, { mockStore })
    );

    expect(page.find("RegistrationForm").length).toEqual(1);
  });

  it("sets the page number to 1", () => {
    const page = mount(<RegistrationPage />);

    expect(page.state()).toMatchObject({ page: 1 });
  });

  it("displays the setup breadcrumbs", () => {
    const mockStore = reduxMockStore(baseStore);
    const page = mount(
      connectedComponent(ConnectedRegistrationPage, { mockStore })
    );

    expect(page.find("Breadcrumbs").length).toEqual(1);
  });

  describe("#onSetPage", () => {
    it("sets state to the page number", () => {
      const page = shallow(<RegistrationPage />);
      page.setState({ page: 3 });
      page.instance().onSetPage(3);

      expect(page.state()).toMatchObject({ page: 3 });
    });
  });
});
