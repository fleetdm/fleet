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

  it("prompts team admins to turn on Apple MDM when it is not configured", () => {
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPremiumTier: true,
          isAnyTeamAdmin: true,
          config: { mdm: { enabled_and_configured: false } } as any,
        },
      },
    });

    render(<AssetsTab currentTeamId={0} router={createMockRouter()} />);

    expect(
      screen.getByRole("button", { name: "Turn on Apple MDM" })
    ).toBeInTheDocument();
  });

  it("does not show the turn on Apple MDM button to technicians", () => {
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPremiumTier: true,
          isGlobalTechnician: true,
          config: { mdm: { enabled_and_configured: false } } as any,
        },
      },
    });

    render(<AssetsTab currentTeamId={0} router={createMockRouter()} />);

    expect(
      screen.getByText("To manage assets, ask your admin to turn on Apple MDM.")
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Turn on Apple MDM" })
    ).not.toBeInTheDocument();
    expect(mdmAPI.getAssets).not.toHaveBeenCalled();
  });

  it("renders the empty state when there are no assets", async () => {
    (mdmAPI.getAssets as jest.Mock).mockResolvedValue({ assets: [] });
    const render = createCustomRenderer({
      withBackendMock: true,
      context: { app: { isPremiumTier: true, config: mdmEnabledConfig } },
    });

    render(<AssetsTab currentTeamId={0} router={createMockRouter()} />);

    expect(await screen.findByText("No assets")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Add asset" })
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
