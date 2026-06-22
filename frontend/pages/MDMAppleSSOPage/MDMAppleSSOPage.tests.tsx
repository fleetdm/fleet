import React from "react";
import { screen, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";

import {
  baseUrl,
  createCustomRenderer,
  createMockRouter,
} from "test/test-utils";
import mockServer from "test/mock-server";
import type { Location as HistoryLocation } from "history";
import { IMDMSSOParams } from "services/entities/mdm";

import DEPSSOLoginPage from "./MDMAppleSSOPage";

const mdmSSOUrl = baseUrl("/mdm/sso");

const render = createCustomRenderer({ withBackendMock: true });

describe("MDMAppleSSOPage", () => {
  const createMockLocation = (
    overrides?: Partial<HistoryLocation<IMDMSSOParams>>
  ): HistoryLocation<IMDMSSOParams> => {
    return {
      action: "PUSH",
      hash: "",
      key: "test-key",
      pathname: "/mdm/sso",
      search: "",
      state: null,
      query: { initiator: "", deviceinfo: "" },
      ...overrides,
    };
  };

  it("calls the SSO API with initiator from the query string when auto-triggered", async () => {
    let requestBody: Record<string, unknown> = {};

    mockServer.use(
      http.post(mdmSSOUrl, async ({ request }) => {
        requestBody = (await request.json()) as Record<string, unknown>;
        return HttpResponse.json({ url: "https://sso.example.com/login" });
      })
    );

    render(
      <DEPSSOLoginPage
        location={createMockLocation({
          pathname: "/mdm/sso",
          query: { initiator: "mdm_sso", deviceinfo: "testdeviceinfo" },
        })}
        params={{}}
        router={createMockRouter()}
        routes={[]}
      />
    );

    await waitFor(() => {
      expect(requestBody.initiator).toBe("mdm_sso");
    });
    expect(requestBody.deviceinfo).toBe("testdeviceinfo");
  });

  it("does not call the SSO API automatically for the setup_experience initiator", async () => {
    const postSpy = jest.fn();

    mockServer.use(
      http.post(mdmSSOUrl, () => {
        postSpy();
        return HttpResponse.json({ url: "https://sso.example.com/login" });
      })
    );

    render(
      <DEPSSOLoginPage
        location={createMockLocation({
          pathname: "/mdm/sso",
          query: {
            initiator: "setup_experience",
            deviceinfo: "testdeviceinfo",
          },
        })}
        params={{}}
        router={createMockRouter()}
        routes={[]}
      />
    );

    // The setup_experience flow shows a sign-in button instead of auto-redirecting
    await waitFor(() => {
      expect(screen.getByText("Sign in")).toBeInTheDocument();
    });

    expect(postSpy).not.toHaveBeenCalled();
  });

  it("fallbacks to mdm_sso initiator when no initiator is provided", async () => {
    let requestBody: Record<string, unknown> = {};

    mockServer.use(
      http.post(mdmSSOUrl, async ({ request }) => {
        requestBody = (await request.json()) as Record<string, unknown>;
        return HttpResponse.json({ url: "https://sso.example.com/login" });
      })
    );

    render(
      <DEPSSOLoginPage
        location={createMockLocation({
          pathname: "/mdm/sso",
          query: { deviceinfo: "testdeviceinfo" },
        })}
        params={{}}
        router={createMockRouter()}
        routes={[]}
      />
    );

    await waitFor(() => {
      expect(requestBody.initiator).toBe("mdm_sso");
    });
    expect(requestBody.deviceinfo).toBe("testdeviceinfo");
  });

  it("uses account_driven_enroll initiator when on the account driven enroll route", async () => {
    let requestBody: Record<string, unknown> = {};

    mockServer.use(
      http.post(mdmSSOUrl, async ({ request }) => {
        requestBody = (await request.json()) as Record<string, unknown>;
        return HttpResponse.json({ url: "https://sso.example.com/login" });
      })
    );

    render(
      <DEPSSOLoginPage
        location={createMockLocation({
          pathname: "/mdm/apple/account_driven_enroll/sso",
          query: { deviceinfo: "testdeviceinfo" },
        })}
        params={{}}
        router={createMockRouter()}
        routes={[]}
      />
    );

    await waitFor(() => {
      expect(requestBody.initiator).toBe("account_driven_enroll");
    });
  });
  it("uses account_driven_enroll initiator with token when on the account driven enroll route with token", async () => {
    let requestBody: Record<string, unknown> = {};

    mockServer.use(
      http.post(mdmSSOUrl, async ({ request }) => {
        requestBody = (await request.json()) as Record<string, unknown>;
        return HttpResponse.json({ url: "https://sso.example.com/login" });
      })
    );

    render(
      <DEPSSOLoginPage
        location={createMockLocation({
          pathname: "/mdm/apple/account_driven_enroll/sso",
          query: { deviceinfo: "testdeviceinfo" },
        })}
        params={{ token: "testtoken" }}
        router={createMockRouter()}
        routes={[]}
      />
    );

    await waitFor(() => {
      expect(requestBody.initiator).toBe("account_driven_enroll:testtoken");
    });
  });
});
