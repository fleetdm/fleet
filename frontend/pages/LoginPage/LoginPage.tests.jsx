import { mount } from "enzyme";

import { connectedComponent, reduxMockStore } from "../../test/helpers";
import LoginPage from "./LoginPage";

const ssoSettings = { sso_enabled: false };

describe("LoginPage - component", () => {
  describe("when the user is not logged in", () => {
    const mockStore = reduxMockStore({ auth: { ssoSettings } });

    it("renders the LoginForm", () => {
      const page = mount(connectedComponent(LoginPage, { mockStore }));

      expect(page.find("LoginForm").length).toEqual(1);
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
      const page = mount(connectedComponent(LoginPage, { mockStore }));
      const loginForm = page.find("LoginForm");

      expect(loginForm.length).toEqual(1);
      expect(loginForm.prop("serverErrors")).toEqual({
        base: "Unable to authenticate the current user",
      });
    });
  });
});
