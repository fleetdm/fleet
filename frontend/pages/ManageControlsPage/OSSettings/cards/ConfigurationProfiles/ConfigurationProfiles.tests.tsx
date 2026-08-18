import React from "react";
import { screen } from "@testing-library/react";
import { http, HttpResponse } from "msw";

import {
  baseUrl,
  createCustomRenderer,
  createMockRouter,
} from "test/test-utils";
import mockServer from "test/mock-server";

import ConfigurationProfiles from "./ConfigurationProfiles";

const emptyProfilesHandler = http.get(baseUrl("/mdm/profiles"), () =>
  HttpResponse.json({
    profiles: [],
    meta: { has_next_results: false, has_previous_results: false },
  })
);

const mdmEnabledConfig = {
  mdm: { enabled_and_configured: true },
} as any;

const mdmDisabledConfig = {
  mdm: {
    enabled_and_configured: false,
    windows_enabled_and_configured: false,
    android_enabled_and_configured: false,
  },
} as any;

const baseProps = {
  currentTeamId: 0,
  router: createMockRouter(),
  onMutation: jest.fn(),
};

describe("ConfigurationProfiles Profiles-tab header", () => {
  it("renders the description and Add profile button when MDM is enabled", async () => {
    mockServer.use(emptyProfilesHandler);

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isGlobalAdmin: true,
          config: mdmEnabledConfig,
        },
      },
    });

    render(<ConfigurationProfiles {...baseProps} />);

    expect(
      await screen.findByText(/Create and upload configuration profiles/i)
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /Add profile$/i })
    ).toBeInTheDocument();
  });

  it("keeps the description visible but hides Add profile when MDM is disabled", async () => {
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isGlobalAdmin: true,
          config: mdmDisabledConfig,
        },
      },
    });

    render(<ConfigurationProfiles {...baseProps} />);

    expect(
      await screen.findByText(/Create and upload configuration profiles/i)
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /Add profile$/i })
    ).not.toBeInTheDocument();
    // The EmptyState below the tab-header still explains why the button is
    // gone, and reads "MDM must be turned on".
    expect(screen.getByText(/MDM must be turned on/i)).toBeInTheDocument();
  });

  it("swaps to the technician description and hides Add profile for technicians", async () => {
    mockServer.use(emptyProfilesHandler);

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isGlobalTechnician: true,
          config: mdmEnabledConfig,
        },
      },
    });

    render(<ConfigurationProfiles {...baseProps} />);

    expect(
      await screen.findByText(/View configuration profiles\./i)
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /Add profile$/i })
    ).not.toBeInTheDocument();
  });

  it("renders the EmptyState heading without Add profile for technicians when there are no profiles", async () => {
    mockServer.use(emptyProfilesHandler);

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isGlobalTechnician: true,
          config: mdmEnabledConfig,
        },
      },
    });

    render(<ConfigurationProfiles {...baseProps} />);

    expect(
      await screen.findByRole("heading", { name: /No configuration profiles/i })
    ).toBeInTheDocument();
    expect(
      screen.getByText(/No configuration profiles have been added\./i)
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /Add profile$/i })
    ).not.toBeInTheDocument();
  });
});
