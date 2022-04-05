import React from "react";
import { render, screen } from "@testing-library/react";

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

    render(connectedComponent(ConnectedRegistrationPage, { mockStore }));

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

  it("displays the Fleet background triangles", () => {
    const mockStore = reduxMockStore(baseStore);

    render(connectedComponent(ConnectedRegistrationPage, { mockStore }));

    expect(mockStore.getActions()).toContainEqual({
      type: "SHOW_BACKGROUND_IMAGE",
    });
  });

  it("does not render the RegistrationForm if the user is loading", () => {
    const mockStore = reduxMockStore({
      app: {},
      auth: { loading: true },
    });

    const { container } = render(
      connectedComponent(ConnectedRegistrationPage, { mockStore })
    );

    expect(container.querySelectorAll(".user-registration").length).toEqual(0);
  });

  it("renders the RegistrationForm when there is no user", () => {
    const mockStore = reduxMockStore(baseStore);

    const { container } = render(
      connectedComponent(ConnectedRegistrationPage, { mockStore })
    );

    expect(container.querySelectorAll(".user-registration").length).toEqual(1);
  });

  it("sets the page number to 1", () => {
    render(<RegistrationPage />);

    expect(
      screen.getByRole("heading", { name: "Setup user" })
    ).toBeInTheDocument();
  });

  it("displays the setup breadcrumbs", () => {
    const mockStore = reduxMockStore(baseStore);
    const { container } = render(
      connectedComponent(ConnectedRegistrationPage, { mockStore })
    );

    expect(
      container.querySelectorAll(".registration-breadcrumbs").length
    ).toEqual(1);
  });
});
