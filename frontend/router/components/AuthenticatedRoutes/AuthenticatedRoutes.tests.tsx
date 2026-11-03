import { http, HttpResponse } from "msw";
import React from "react";

import createMockUser from "__mocks__/userMock";
import { RoutingContext } from "context/routing";
import paths from "router/paths";
import usersAPI from "services/entities/users";
import mockServer from "test/mock-server";
import {
  baseUrl,
  createCustomRenderer,
  createMockRouter,
} from "test/test-utils";
import authToken from "utilities/auth_token";

import AuthenticatedRoutes from "./AuthenticatedRoutes";

const expiredSessionHandler = http.get(baseUrl("/me"), () =>
  HttpResponse.json({ message: "Authentication required" }, { status: 401 })
);

const renderRoutes = (pathname: string, search = "") => {
  const router = createMockRouter();
  const setRedirectLocation = jest.fn();
  const render = createCustomRenderer({
    context: { app: { currentUser: createMockUser() } },
  });
  render(
    <RoutingContext.Provider
      value={{ redirectLocation: null, setRedirectLocation }}
    >
      <AuthenticatedRoutes
        router={router}
        location={{ pathname, search, hash: "" }}
      >
        <div>Hosts page</div>
      </AuthenticatedRoutes>
    </RoutingContext.Provider>
  );
  return { router, setRedirectLocation };
};

describe("AuthenticatedRoutes", () => {
  beforeEach(() => {
    jest.spyOn(window, "scrollTo").mockImplementation(() => undefined);
    authToken.save("session-token");
  });

  afterEach(() => {
    authToken.remove();
    jest.restoreAllMocks();
  });

  it("sends the user to the login page and back to where they were when their session expires", async () => {
    mockServer.use(expiredSessionHandler);
    const { router, setRedirectLocation } = renderRoutes(
      "/hosts/manage",
      "?fleet_id=3"
    );

    await expect(usersAPI.me()).rejects.toBeDefined();

    expect(router.push).toHaveBeenCalledWith(
      `${paths.LOGIN}?session_expired=true`
    );
    expect(setRedirectLocation).toHaveBeenCalledWith(
      "/hosts/manage?fleet_id=3"
    );
    expect(authToken.get()).toBeNull();
  });

  it("redirects once when several requests fail with an expired session", async () => {
    mockServer.use(expiredSessionHandler);
    const { router } = renderRoutes("/hosts/manage");

    await Promise.allSettled([usersAPI.me(), usersAPI.me()]);

    expect(router.push).toHaveBeenCalledTimes(1);
  });

  it("leaves a failed logout to the logout page", async () => {
    mockServer.use(expiredSessionHandler);
    const { router } = renderRoutes(paths.LOGOUT);

    await expect(usersAPI.me()).rejects.toBeDefined();

    expect(router.push).not.toHaveBeenCalled();
  });

  it("does not redirect for errors other than an expired session", async () => {
    mockServer.use(
      http.get(baseUrl("/me"), () =>
        HttpResponse.json({ message: "Forbidden" }, { status: 403 })
      )
    );
    const { router } = renderRoutes("/hosts/manage");

    await expect(usersAPI.me()).rejects.toBeDefined();

    expect(router.push).not.toHaveBeenCalled();
    expect(authToken.get()).toBe("session-token");
  });
});
