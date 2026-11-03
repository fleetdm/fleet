import { screen, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import React from "react";
import { QueryClient, QueryClientProvider } from "react-query";

import { createMockConfig } from "__mocks__/configMock";
import createMockUser from "__mocks__/userMock";
import paths from "router/paths";
import mockServer from "test/mock-server";
import {
  baseUrl,
  createCustomRenderer,
  createMockLocation,
  createMockRouter,
} from "test/test-utils";
import authToken from "utilities/auth_token";

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

  afterEach(() => {
    authToken.remove();
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

  it("drops the previous user's cached data after logging in", async () => {
    mockServer.use(
      http.post(baseUrl("/login"), () =>
        HttpResponse.json({
          user: createMockUser({ id: 2, email: "next@example.com" }),
          available_teams: [],
          token: "next-user-token",
        })
      )
    );
    const queryClient = new QueryClient();
    queryClient.setQueryData([1, 7], { report: "previous user's results" });
    const router = createMockRouter();
    const render = createCustomRenderer({
      context: {
        app: {
          config: createMockConfig(),
          setCurrentUser: jest.fn(),
          setAvailableTeams: jest.fn(),
          setCurrentTeam: jest.fn(),
        },
      },
    });
    const { user } = render(
      <QueryClientProvider client={queryClient}>
        <LoginPage
          router={router}
          location={createMockLocation({ pathname: "/login" })}
        />
      </QueryClientProvider>
    );

    await user.type(await screen.findByLabelText("Email"), "next@example.com");
    await user.type(screen.getByLabelText("Password"), "password");
    await user.click(screen.getByRole("button", { name: "Log in" }));

    await waitFor(() => {
      expect(router.push).toHaveBeenCalledWith(paths.DASHBOARD);
    });
    expect(queryClient.getQueryData([1, 7])).toBeUndefined();
  });
});
