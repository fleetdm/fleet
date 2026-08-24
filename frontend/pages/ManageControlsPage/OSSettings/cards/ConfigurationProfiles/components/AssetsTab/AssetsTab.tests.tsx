import React from "react";
import { screen } from "@testing-library/react";

import PATHS from "router/paths";
import { createCustomRenderer, createMockRouter } from "test/test-utils";
import mdmAPI from "services/entities/mdm";

import AssetsTab from "./AssetsTab";

jest.mock("services/entities/mdm", () => ({
  __esModule: true,
  default: {
    getAssets: jest.fn(),
    deleteAsset: jest.fn(),
    downloadAsset: jest.fn(),
    uploadAsset: jest.fn(),
  },
}));

const mdmEnabledConfig = {
  mdm: { enabled_and_configured: true },
} as any;

describe("AssetsTab", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("shows the premium message on Fleet Free", () => {
    const render = createCustomRenderer({
      withBackendMock: true,
      context: { app: { isPremiumTier: false, config: mdmEnabledConfig } },
    });

    render(<AssetsTab currentTeamId={0} router={createMockRouter()} />);

    expect(
      screen.getByText(/This feature is included in Fleet Premium/i)
    ).toBeInTheDocument();
    expect(mdmAPI.getAssets).not.toHaveBeenCalled();
  });

  it("prompts global admins to turn on Apple MDM when it is not configured", async () => {
    const router = createMockRouter();
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPremiumTier: true,
          isGlobalAdmin: true,
          config: { mdm: { enabled_and_configured: false } } as any,
        },
      },
    });

    const { user } = render(<AssetsTab currentTeamId={0} router={router} />);

    const button = screen.getByRole("button", { name: "Turn on Apple MDM" });
    await user.click(button);
    expect(router.push).toHaveBeenCalledWith(
      PATHS.ADMIN_INTEGRATIONS_MDM_APPLE
    );
    expect(mdmAPI.getAssets).not.toHaveBeenCalled();
  });

  it("does not show the turn on Apple MDM button to non-global-admins", () => {
    const nonGlobalAdminContexts = [
      { isAnyTeamAdmin: true },
      { isGlobalTechnician: true },
      { isTeamTechnician: true },
    ];

    nonGlobalAdminContexts.forEach((roleContext) => {
      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            isPremiumTier: true,
            ...roleContext,
            config: { mdm: { enabled_and_configured: false } } as any,
          },
        },
      });

      const { unmount } = render(
        <AssetsTab currentTeamId={0} router={createMockRouter()} />
      );

      expect(
        screen.getByText("Supported on macOS, iOS, and iPadOS.")
      ).toBeInTheDocument();
      expect(
        screen.queryByRole("button", { name: "Turn on Apple MDM" })
      ).not.toBeInTheDocument();
      expect(mdmAPI.getAssets).not.toHaveBeenCalled();

      unmount();
    });
  });

  it("renders the empty state when there are no assets", async () => {
    (mdmAPI.getAssets as jest.Mock).mockResolvedValue({ assets: [] });
    const render = createCustomRenderer({
      withBackendMock: true,
      context: { app: { isPremiumTier: true, config: mdmEnabledConfig } },
    });

    render(<AssetsTab currentTeamId={0} router={createMockRouter()} />);

    expect(await screen.findByText("No assets")).toBeInTheDocument();
    // Empty state's own "Add asset" primaryButton + the persistent tab-header
    // "Add asset" (accessible name "plus Add asset" from its icon) — both
    // match /Add asset$/i.
    expect(screen.getAllByRole("button", { name: /Add asset$/i })).toHaveLength(
      2
    );
  });

  it("renders the EmptyState heading without Add asset for technicians when there are no assets", async () => {
    (mdmAPI.getAssets as jest.Mock).mockResolvedValue({ assets: [] });
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPremiumTier: true,
          isGlobalTechnician: true,
          config: mdmEnabledConfig,
        },
      },
    });

    render(<AssetsTab currentTeamId={0} router={createMockRouter()} />);

    expect(
      await screen.findByRole("heading", { name: /No assets/i })
    ).toBeInTheDocument();
    expect(
      screen.getByText(/No assets have been added\./i)
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /Add asset$/i })
    ).not.toBeInTheDocument();
  });

  it("renders the tab-header description and Add asset button above the list", async () => {
    (mdmAPI.getAssets as jest.Mock).mockResolvedValue({
      assets: [
        {
          asset_uuid: "u1",
          name: "Asset",
          identifier: "com.example.asset1",
          created_at: "2024-01-01T00:00:00Z",
          uploaded_at: "2024-01-01T00:00:00Z",
          checksum: "abc",
        },
      ],
    });
    const render = createCustomRenderer({
      withBackendMock: true,
      context: { app: { isPremiumTier: true, config: mdmEnabledConfig } },
    });

    render(<AssetsTab currentTeamId={0} router={createMockRouter()} />);

    expect(
      screen.getByText(/Manage assets that provide data or credentials/i)
    ).toBeInTheDocument();
    expect(
      await screen.findByRole("button", { name: /Add asset$/i })
    ).toBeInTheDocument();
  });

  it("renders the list of assets", async () => {
    (mdmAPI.getAssets as jest.Mock).mockResolvedValue({
      assets: [
        {
          asset_uuid: "u1",
          name: "JSON Asset",
          identifier: "com.example.asset1",
          created_at: "2024-01-01T00:00:00Z",
          uploaded_at: "2024-01-01T00:00:00Z",
          checksum: "abc",
        },
      ],
    });
    const render = createCustomRenderer({
      withBackendMock: true,
      context: { app: { isPremiumTier: true, config: mdmEnabledConfig } },
    });

    render(<AssetsTab currentTeamId={0} router={createMockRouter()} />);

    expect(await screen.findByText("com.example.asset1")).toBeInTheDocument();
    expect(mdmAPI.getAssets).toHaveBeenCalledWith({ fleet_id: 0 });
  });
});
