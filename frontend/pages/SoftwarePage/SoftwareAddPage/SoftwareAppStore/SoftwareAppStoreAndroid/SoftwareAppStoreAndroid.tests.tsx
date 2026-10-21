import { screen } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import React from "react";

import mockServer from "test/mock-server";
import { createCustomRenderer, createMockRouter } from "test/test-utils";

import SoftwareAppStoreAndroid from "./SoftwareAppStoreAndroid";

const baseUrl = (path: string) => `/api/latest/fleet${path}`;

const router = createMockRouter();

describe("SoftwareAppStoreAndroid", () => {
  afterEach(() => {
    mockServer.resetHandlers();
  });

  it("shows enable button for admins when Android MDM is not configured", () => {
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPremiumTier: true,
          isGlobalAdmin: true,
          isAndroidMdmEnabledAndConfigured: false,
        },
      },
    });

    render(<SoftwareAppStoreAndroid currentTeamId={1} router={router} />);

    expect(screen.getByText("Android MDM isn't enabled")).toBeInTheDocument();
    expect(
      screen.getByText("To add Android apps, first enable Android MDM.")
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Enable Android MDM" })
    ).toBeInTheDocument();
  });

  it("shows ask your admin copy for non-admins when Android MDM is not configured", () => {
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPremiumTier: true,
          isGlobalAdmin: false,
          isAndroidMdmEnabledAndConfigured: false,
        },
      },
    });

    render(<SoftwareAppStoreAndroid currentTeamId={1} router={router} />);

    expect(screen.getByText("Android MDM isn't enabled")).toBeInTheDocument();
    expect(
      screen.getByText(
        "To add Android apps, ask your admin to enable Android MDM."
      )
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Enable Android MDM" })
    ).not.toBeInTheDocument();
  });

  it("shows the Android form with the fleet's custom labels as targets when Android MDM is enabled", async () => {
    let requestedFleetId: string | null = null;
    mockServer.use(
      http.get(baseUrl("/labels/summary"), ({ request }) => {
        requestedFleetId = new URL(request.url).searchParams.get("fleet_id");
        return HttpResponse.json({
          labels: [
            { id: 1, name: "All Hosts", label_type: "builtin" },
            { id: 2, name: "Android pilot", label_type: "regular" },
          ],
        });
      })
    );
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPremiumTier: true,
          isGlobalAdmin: true,
          isAndroidMdmEnabledAndConfigured: true,
        },
      },
    });

    const { user } = render(
      <SoftwareAppStoreAndroid currentTeamId={1} router={router} />
    );

    await user.click(await screen.findByRole("radio", { name: "Custom" }));

    expect(
      screen.getByRole("checkbox", { name: "Android pilot" })
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("checkbox", { name: "All Hosts" })
    ).not.toBeInTheDocument();
    expect(requestedFleetId).toBe("1");
    expect(
      screen.queryByText("Android MDM isn't enabled")
    ).not.toBeInTheDocument();
  });

  it("requests global labels for the Unassigned fleet", async () => {
    let requestedFleetId: string | null = null;
    mockServer.use(
      http.get(baseUrl("/labels/summary"), ({ request }) => {
        requestedFleetId = new URL(request.url).searchParams.get("fleet_id");
        return HttpResponse.json({
          labels: [{ id: 2, name: "Android pilot", label_type: "regular" }],
        });
      })
    );
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPremiumTier: true,
          isGlobalAdmin: true,
          isAndroidMdmEnabledAndConfigured: true,
        },
      },
    });

    const { user } = render(
      <SoftwareAppStoreAndroid currentTeamId={0} router={router} />
    );

    await user.click(await screen.findByRole("radio", { name: "Custom" }));

    expect(
      screen.getByRole("checkbox", { name: "Android pilot" })
    ).toBeInTheDocument();
    expect(requestedFleetId).toBe("global");
  });

  it("waits for custom labels before showing the form", async () => {
    mockServer.use(
      http.get(baseUrl("/labels/summary"), () =>
        HttpResponse.json({ labels: [] })
      )
    );
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPremiumTier: true,
          isGlobalAdmin: true,
          isAndroidMdmEnabledAndConfigured: true,
        },
      },
    });

    render(<SoftwareAppStoreAndroid currentTeamId={1} router={router} />);

    expect(
      screen.queryByRole("textbox", { name: "Application ID" })
    ).not.toBeInTheDocument();
    expect(
      await screen.findByRole("textbox", { name: "Application ID" })
    ).toBeInTheDocument();
  });

  it("shows an error when custom labels fail to load", async () => {
    mockServer.use(
      http.get(baseUrl("/labels/summary"), () =>
        HttpResponse.json({}, { status: 404 })
      )
    );
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPremiumTier: true,
          isGlobalAdmin: true,
          isAndroidMdmEnabledAndConfigured: true,
        },
      },
    });

    render(<SoftwareAppStoreAndroid currentTeamId={1} router={router} />);

    expect(
      await screen.findByText(/Something's gone wrong/)
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Add software" })
    ).not.toBeInTheDocument();
  });
});
