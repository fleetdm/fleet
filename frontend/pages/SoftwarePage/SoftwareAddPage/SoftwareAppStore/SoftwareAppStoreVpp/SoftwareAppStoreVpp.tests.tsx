import React from "react";
import { screen, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";

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

describe("SoftwareAppStoreVpp", () => {
  afterEach(() => {
    mockServer.resetHandlers();
  });

  it("shows enable VPP button for admins when no VPP tokens exist", async () => {
    mockServer.use(emptyVppHandler, labelsHandler);

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPremiumTier: true,
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
          isGlobalAdmin: false,
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
      screen.getByText(
        "To add App Store apps, ask your admin to add this fleet to VPP."
      )
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Edit VPP" })
    ).not.toBeInTheDocument();
  });
});
