import { screen, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import React from "react";

import mockServer from "test/mock-server";
import { createCustomRenderer, createMockRouter } from "test/test-utils";

import SoftwareAppStoreVpp from "./SoftwareAppStoreVpp";

const baseUrl = (path: string) => `/api/latest/fleet${path}`;

const router = createMockRouter();

const emptyVppHandler = http.get(baseUrl("/vpp_tokens"), () => {
  return HttpResponse.json({ vpp_tokens: [] });
});

const labelsHandler = http.get(baseUrl("/labels/summary"), () => {
  return HttpResponse.json({ labels: [] });
});

const teamMismatchVppHandler = http.get(baseUrl("/vpp_tokens"), () => {
  return HttpResponse.json({
    vpp_tokens: [
      {
        id: 1,
        org_name: "Test Org",
        location: "US",
        renew_date: "2027-01-01",
        teams: [{ team_id: 999, name: "Other fleet" }],
      },
    ],
  });
});

const teamVppHandler = http.get(baseUrl("/vpp_tokens"), () => {
  return HttpResponse.json({
    vpp_tokens: [
      {
        id: 1,
        org_name: "Test Org",
        location: "US",
        renew_date: "2027-01-01",
        teams: [{ team_id: 1, name: "Workstations" }],
      },
    ],
  });
});

describe("SoftwareAppStoreVpp", () => {
  afterEach(() => {
    mockServer.resetHandlers();
  });

  it("shows enable VPP when Apple MDM is off even if the fleet has a VPP token", async () => {
    let appsRequested = false;
    mockServer.use(
      teamVppHandler,
      labelsHandler,
      http.get(baseUrl("/software/app_store_apps"), () => {
        appsRequested = true;
        return HttpResponse.json({ app_store_apps: [] });
      })
    );

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPremiumTier: true,
          isGlobalAdmin: true,
          isMacMdmEnabledAndConfigured: false,
        },
      },
    });

    render(<SoftwareAppStoreVpp currentTeamId={1} router={router} />);

    await waitFor(() => {
      expect(
        screen.getByText(/Volume Purchasing Program \(VPP\) isn't enabled/i)
      ).toBeInTheDocument();
    });
    expect(
      screen.getByText("Enable VPP to add App Store apps (MDM required).")
    ).toBeInTheDocument();
    expect(appsRequested).toBe(false);
  });

  it("shows enable VPP button for admins when no VPP tokens exist", async () => {
    mockServer.use(emptyVppHandler, labelsHandler);

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPremiumTier: true,
          isMacMdmEnabledAndConfigured: true,
          isGlobalAdmin: true,
        },
      },
    });

    render(<SoftwareAppStoreVpp currentTeamId={1} router={router} />);

    await waitFor(() => {
      expect(
        screen.getByText(/Volume Purchasing Program \(VPP\) isn't enabled/i)
      ).toBeInTheDocument();
    });

    expect(
      screen.getByText("Enable VPP to add App Store apps (MDM required).")
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Enable VPP" })
    ).toBeInTheDocument();
  });

  it("shows ask your admin copy for non-admins when no VPP tokens exist", async () => {
    mockServer.use(emptyVppHandler, labelsHandler);

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPremiumTier: true,
          isMacMdmEnabledAndConfigured: true,
          isGlobalAdmin: false,
        },
      },
    });

    render(<SoftwareAppStoreVpp currentTeamId={1} router={router} />);

    await waitFor(() => {
      expect(
        screen.getByText(/Volume Purchasing Program \(VPP\) isn't enabled/i)
      ).toBeInTheDocument();
    });

    expect(
      screen.getByText("To add App Store apps, ask your admin to enable VPP.")
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Enable VPP" })
    ).not.toBeInTheDocument();
  });

  it("shows add fleet to VPP button for admins when team has no VPP token", async () => {
    mockServer.use(teamMismatchVppHandler, labelsHandler);

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPremiumTier: true,
          isMacMdmEnabledAndConfigured: true,
          isGlobalAdmin: true,
        },
      },
    });

    render(
      <SoftwareAppStoreVpp
        currentTeamId={1}
        currentTeamName="Workstations"
        router={router}
      />
    );

    await waitFor(() => {
      expect(
        screen.getByText(
          /Workstations fleet isn't added to Volume Purchasing Program \(VPP\)/i
        )
      ).toBeInTheDocument();
    });

    expect(
      screen.getByText(
        "To add App Store apps, first add Workstations fleet to VPP."
      )
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Edit VPP" })
    ).toBeInTheDocument();
  });

  it("shows ask your admin copy for non-admins when team has no VPP token", async () => {
    mockServer.use(teamMismatchVppHandler, labelsHandler);

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPremiumTier: true,
          isMacMdmEnabledAndConfigured: true,
          isGlobalAdmin: false,
        },
      },
    });

    render(
      <SoftwareAppStoreVpp
        currentTeamId={1}
        currentTeamName="Workstations"
        router={router}
      />
    );

    await waitFor(() => {
      expect(
        screen.getByText(
          /Workstations fleet isn't added to Volume Purchasing Program \(VPP\)/i
        )
      ).toBeInTheDocument();
    });

    expect(
      screen.getByText(
        "To add App Store apps, ask your admin to add Workstations fleet to VPP."
      )
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Edit VPP" })
    ).not.toBeInTheDocument();
  });

  it("shows unassigned-hosts copy when No team has no VPP token", async () => {
    mockServer.use(teamMismatchVppHandler, labelsHandler);

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPremiumTier: true,
          isMacMdmEnabledAndConfigured: true,
          isGlobalAdmin: true,
        },
      },
    });

    render(
      <SoftwareAppStoreVpp
        currentTeamId={0}
        currentTeamName="No team"
        router={router}
      />
    );

    await waitFor(() => {
      expect(
        screen.getByText(
          /Unassigned hosts aren't in Volume Purchasing Program \(VPP\)/i
        )
      ).toBeInTheDocument();
    });

    expect(
      screen.getByText(
        "To add App Store apps, first add unassigned hosts to VPP."
      )
    ).toBeInTheDocument();
  });

  it("falls back to generic copy when the fleet name hasn't resolved yet", async () => {
    mockServer.use(teamMismatchVppHandler, labelsHandler);

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPremiumTier: true,
          isMacMdmEnabledAndConfigured: true,
          isGlobalAdmin: true,
        },
      },
    });

    render(<SoftwareAppStoreVpp currentTeamId={1} router={router} />);

    await waitFor(() => {
      expect(
        screen.getByText(
          /This fleet isn't added to Volume Purchasing Program \(VPP\)/i
        )
      ).toBeInTheDocument();
    });

    expect(
      screen.getByText("To add App Store apps, first add this fleet to VPP.")
    ).toBeInTheDocument();
  });
});
