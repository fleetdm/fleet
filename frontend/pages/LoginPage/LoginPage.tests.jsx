import { render, screen } from "@testing-library/react";

import { connectedComponent, reduxMockStore } from "../../test/helpers";
import LoginPage from "./LoginPage";

const ssoSettings = { sso_enabled: false };

describe("LoginPage - component", () => {
  describe("when the user is not logged in", () => {
    const mockStore = reduxMockStore({ auth: { ssoSettings } });

    it("renders the LoginForm", () => {
      const { container } = render(
        connectedComponent(LoginPage, { mockStore })
      );

      expect(container.querySelectorAll(".login-form").length).toEqual(1);
    });
  });

  describe("when the users session is not recognized", () => {
    const mockStore = reduxMockStore({
      auth: {
        errors: { base: "Unable to authenticate the current user" },
        ssoSettings,
      },
    });

    it("renders the LoginForm base errors", () => {
      const { container } = render(
        connectedComponent(LoginPage, { mockStore })
      );
      const loginForm = container.querySelectorAll(".login-form");

      expect(loginForm.length).toEqual(1);
      expect(
        screen.getByText("Unable to authenticate the current user")
      ).toBeInTheDocument();
    });
  });
});
