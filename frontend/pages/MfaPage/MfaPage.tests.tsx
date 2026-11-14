import { render, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import React from "react";

import { createMockConfig } from "__mocks__/configMock";
import createMockUser from "__mocks__/userMock";
import AppProvider from "context/app";
import paths from "router/paths";
import mockServer from "test/mock-server";
import { baseUrl, createMockRouter } from "test/test-utils";
import authToken from "utilities/auth_token";
import local from "utilities/local";

import MfaPage from "./MfaPage";

describe("MfaPage", () => {
  afterEach(() => {
    authToken.remove();
    local.removeItem("auth_pending_mfa");
  });

  it("redeems the single-use MFA token only once", async () => {
    const mfaRequestBodies: unknown[] = [];
    mockServer.use(
      http.post(baseUrl("/sessions"), async ({ request }) => {
        mfaRequestBodies.push(await request.json());
        if (mfaRequestBodies.length > 1) {
          return HttpResponse.json(
            { message: "Authentication failed" },
            { status: 401 }
          );
        }
        return HttpResponse.json({
          user: createMockUser(),
          available_fleets: [],
          token: "new-session-token",
        });
      }),
      http.get(baseUrl("/config"), () => HttpResponse.json(createMockConfig()))
    );
    local.setItem("auth_pending_mfa", "true");
    const router = createMockRouter();

    // Uses the real AppProvider: the bug is triggered by the context update
    // after a successful redemption re-rendering the page.
    render(
      <AppProvider>
        <MfaPage router={router} params={{ token: "magic-link-token" }} />
      </AppProvider>
    );

    await waitFor(() => {
      expect(router.push).toHaveBeenCalledWith(paths.DASHBOARD);
    });
    // Let any re-render-triggered request reach the mock server.
    await new Promise((resolve) => setTimeout(resolve, 50));

    expect(mfaRequestBodies).toEqual([{ token: "magic-link-token" }]);
    expect(authToken.get()).toBe("new-session-token");
  });
});
