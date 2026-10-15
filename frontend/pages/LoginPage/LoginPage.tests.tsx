import { screen } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import React from "react";

import createMockUser from "__mocks__/userMock";
import mockServer from "test/mock-server";
import {
  createCustomRenderer,
  createMockLocation,
  createMockRouter,
} from "test/test-utils";

import LoginPage from "./LoginPage";

const ssoEnabledHandler = http.get("/api/v1/fleet/sso", () =>
  HttpResponse.json({ settings: { sso_enabled: true } })
);

const gravatarHandler = http.get(
  "https://www.gravatar.com/avatar/0000",
  () => new HttpResponse(null, { status: 404 })
);

const renderLoginPage = (
  query: Record<string, string> = {},
  currentUser = null as ReturnType<typeof createMockUser> | null
) => {
  const render = createCustomRenderer({
    withBackendMock: true,
    context: { app: { currentUser } },
  });
  return render(
    <LoginPage
      router={createMockRouter()}
      location={createMockLocation({ pathname: "/login", query })}
    />
  );
};

describe("LoginPage", () => {
  beforeEach(() => {
    mockServer.use(ssoEnabledHandler, gravatarHandler);
  });

  it("tells the user their session expired", async () => {
    renderLoginPage({ session_expired: "true" });

    expect(
      await screen.findByText("Your session expired. Log in to continue.")
    ).toBeInTheDocument();
  });

  it("does not show the session-expired notice on a regular visit", async () => {
    renderLoginPage();

    await screen.findByRole("button", { name: "Log in" });
    expect(
      screen.queryByText("Your session expired. Log in to continue.")
    ).not.toBeInTheDocument();
  });

  it("offers SSO when the expired session's user is still in app context", async () => {
    renderLoginPage({ session_expired: "true" }, createMockUser());

    expect(
      await screen.findByRole("button", { name: /sign in with sso/i })
    ).toBeInTheDocument();
  });
});
