import React from "react";

import { screen } from "@testing-library/react";

import { createMockRouter, createCustomRenderer } from "test/test-utils";
import { createMockConfig, createMockMdmConfig } from "__mocks__/configMock";
import configAPI from "services/entities/config";
import WindowsMdmPage from "./WindowsMdmPage";

jest.mock("services/entities/config", () => ({
  updateMDMConfig: jest.fn(() => Promise.resolve({})),
}));

describe("WindowsMdmPage", () => {
  it("renders only the windows mdm slider when on free tier", () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isPremiumTier: false,
          config: createMockConfig(),
        },
      },
    });

    render(<WindowsMdmPage router={createMockRouter()} />);

    expect(screen.getByRole("switch")).toBeInTheDocument();

    // no premium-only sections
    expect(
      screen.queryByText("Turn on MDM programmatically")
    ).not.toBeInTheDocument();
    expect(
      screen.queryByText("User driven enrollment")
    ).not.toBeInTheDocument();
    expect(screen.queryByText("Migration")).not.toBeInTheDocument();
  });

  it("renders the programmatic enrollment toggle as disabled when MDM is off", () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isPremiumTier: true,
          config: createMockConfig({
            mdm: createMockMdmConfig({ windows_enabled_and_configured: false }),
          }),
        },
      },
    });

    render(<WindowsMdmPage router={createMockRouter()} />);

    expect(screen.getByText("Turn on MDM programmatically")).toBeVisible();
    const switches = screen.getAllByRole("switch");
    expect(switches[1]).toBeDisabled();
  });

  it("renders the Migration section when MDM is on programmatically", () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isPremiumTier: true,
          config: createMockConfig({
            mdm: createMockMdmConfig({
              enable_turn_on_windows_mdm_manually: false,
              windows_enabled_and_configured: true,
            }),
          }),
        },
      },
    });

    render(<WindowsMdmPage router={createMockRouter()} />);

    expect(screen.getByText("Migration")).toBeVisible();
    expect(screen.getByRole("checkbox")).toBeVisible();
  });

  it("disables the default fleet dropdown when Fleet is not connected to Entra", () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isPremiumTier: true,
          config: createMockConfig({
            mdm: createMockMdmConfig({
              windows_enabled_and_configured: true,
              windows_entra_tenant_ids: [],
            }),
          }),
        },
      },
    });

    render(<WindowsMdmPage router={createMockRouter()} />);

    expect(screen.getByText("User driven enrollment")).toBeVisible();
    expect(screen.getByText("Default fleet")).toBeVisible();
    expect(screen.getByRole("combobox")).toBeDisabled();
  });

  it("enables the default fleet dropdown when Fleet is connected to Entra", () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isPremiumTier: true,
          config: createMockConfig({
            mdm: createMockMdmConfig({
              windows_enabled_and_configured: true,
              windows_entra_tenant_ids: ["tenant-1"],
            }),
          }),
        },
      },
    });

    render(<WindowsMdmPage router={createMockRouter()} />);

    expect(screen.getByRole("combobox")).toBeEnabled();
  });

  it("saves the toggle states and the default fleet through the config API", async () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isPremiumTier: true,
          config: createMockConfig({
            mdm: createMockMdmConfig({
              windows_enabled_and_configured: true,
              enable_turn_on_windows_mdm_manually: false,
              windows_entra_tenant_ids: ["tenant-1"],
              windows_enrollment: { default_fleet: "Workstations" },
            }),
          }),
        },
      },
    });

    const { user } = render(<WindowsMdmPage router={createMockRouter()} />);

    // Turning programmatic enrollment off also forces auto migration off.
    const switches = screen.getAllByRole("switch");
    await user.click(switches[1]);
    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(configAPI.updateMDMConfig).toHaveBeenCalledWith(
      {
        windows_enabled_and_configured: true,
        enable_turn_on_windows_mdm_manually: true,
        windows_migration_enabled: false,
        windows_enrollment: { default_fleet: "Workstations" },
      },
      true
    );
  });
});
