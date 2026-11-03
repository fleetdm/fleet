import { screen } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import React from "react";

import mockServer from "test/mock-server";
import {
  baseUrl,
  createCustomRenderer,
  createMockRouter,
} from "test/test-utils";

import ConfigurationProfiles from "./ConfigurationProfiles";

const emptyProfilesHandler = http.get(baseUrl("/mdm/profiles"), () =>
  HttpResponse.json({
    profiles: [],
    meta: { has_next_results: false, has_previous_results: false },
  })
);

const emptyAssetsHandler = http.get(baseUrl("/assets"), () =>
  HttpResponse.json({ assets: [] })
);

const asset = {
  asset_uuid: "a-123",
  name: "Wi-Fi certificate",
  identifier: "com.example.asset1",
  created_at: "2024-01-01T00:00:00Z",
  uploaded_at: "2024-01-01T00:00:00Z",
  checksum: "abc",
};

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

describe("ConfigurationProfiles header", () => {
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

describe("ConfigurationProfiles add and edit", () => {
  const profile = {
    profile_uuid: "w-123",
    team_id: 2,
    name: "Firewall",
    platform: "windows",
    identifier: null,
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
    checksum: null,
  };

  const renderList = () => {
    const router = createMockRouter();
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPremiumTier: true,
          isGlobalAdmin: true,
          config: mdmEnabledConfig,
        },
      },
    });
    const results = render(
      <ConfigurationProfiles {...baseProps} currentTeamId={2} router={router} />
    );
    return { ...results, router };
  };

  it("opens the add page for the current fleet", async () => {
    mockServer.use(emptyProfilesHandler, emptyAssetsHandler);
    const { user, router } = renderList();

    await user.click(
      await screen.findByRole("button", { name: /Add profile$/i })
    );

    expect(router.push).toHaveBeenCalledTimes(1);
    expect(router.push).toHaveBeenCalledWith(
      "/controls/os-settings/configuration-profiles/new?fleet_id=2"
    );
  });

  it("opens a profile's edit page for the current fleet", async () => {
    mockServer.use(
      http.get(baseUrl("/mdm/profiles"), () =>
        HttpResponse.json({
          profiles: [profile],
          meta: { has_next_results: false, has_previous_results: false },
        })
      ),
      emptyAssetsHandler
    );
    const { user, router } = renderList();

    await user.click(await screen.findByLabelText("Edit Firewall"));

    expect(router.push).toHaveBeenCalledTimes(1);
    expect(router.push).toHaveBeenCalledWith(
      "/controls/os-settings/configuration-profiles/w-123?fleet_id=2"
    );
  });
});

describe("ConfigurationProfiles assets", () => {
  const renderPremium = (app: Record<string, unknown> = {}) => {
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPremiumTier: true,
          isGlobalAdmin: true,
          config: mdmEnabledConfig,
          ...app,
        },
      },
    });
    return render(<ConfigurationProfiles {...baseProps} />);
  };

  it("lists assets alongside profiles in a single list", async () => {
    mockServer.use(
      http.get(baseUrl("/mdm/profiles"), () =>
        HttpResponse.json({
          profiles: [
            {
              profile_uuid: "p-1",
              team_id: 0,
              name: "Firewall",
              platform: "windows",
              identifier: null,
              created_at: "2024-01-01T00:00:00Z",
              updated_at: "2024-01-01T00:00:00Z",
              checksum: null,
            },
          ],
          meta: { has_next_results: false, has_previous_results: false },
        })
      ),
      http.get(baseUrl("/assets"), () => HttpResponse.json({ assets: [asset] }))
    );

    renderPremium();

    expect(await screen.findByText("com.example.asset1")).toBeInTheDocument();
    expect(screen.getByText("Asset")).toBeInTheDocument();
    expect(screen.getByText("Firewall")).toBeInTheDocument();
    expect(screen.queryByRole("tab")).not.toBeInTheDocument();
  });

  it("opens the Add asset modal from the Advanced button", async () => {
    mockServer.use(emptyProfilesHandler, emptyAssetsHandler);

    const { user } = renderPremium();

    await user.click(await screen.findByRole("button", { name: "Advanced" }));

    expect(
      screen.getByText(/Assets are reusable data or credentials/i)
    ).toBeInTheDocument();
  });

  it("hides the Advanced button on Fleet Free", async () => {
    mockServer.use(emptyProfilesHandler);

    renderPremium({ isPremiumTier: false });

    expect(
      await screen.findByRole("button", { name: /Add profile$/i })
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Advanced" })
    ).not.toBeInTheDocument();
  });

  it("hides the Advanced button for technicians", async () => {
    mockServer.use(emptyProfilesHandler, emptyAssetsHandler);

    renderPremium({ isGlobalAdmin: false, isGlobalTechnician: true });

    expect(
      await screen.findByText(/View configuration profiles\./i)
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Advanced" })
    ).not.toBeInTheDocument();
  });
});
