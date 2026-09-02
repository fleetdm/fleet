import React from "react";

import { screen } from "@testing-library/react";

import { createMockRouter, createCustomRenderer } from "test/test-utils";
import { createMockConfig, createMockMdmConfig } from "__mocks__/configMock";
import { IMdmConfig } from "interfaces/config";
import configAPI from "services/entities/config";
import WindowsMdmPage from "./WindowsMdmPage";

jest.mock("services/entities/config");

const renderPage = (mdm: Partial<IMdmConfig> = {}, isPremiumTier = true) => {
  const render = createCustomRenderer({
    context: {
      app: {
        isPremiumTier,
        config: createMockConfig({ mdm: createMockMdmConfig(mdm) }),
      },
    },
  });

  return render(<WindowsMdmPage router={createMockRouter()} />);
};

describe("WindowsMdmPage", () => {
  it("renders only the windows mdm slider when on free tier", () => {
    renderPage({}, false);

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
    renderPage({ windows_enabled_and_configured: false });

    expect(screen.getByText("Turn on MDM programmatically")).toBeVisible();
    expect(screen.getAllByRole("switch")[1]).toBeDisabled();
  });

  it("renders the Migration section when MDM is on programmatically", () => {
    renderPage({
      enable_turn_on_windows_mdm_manually: false,
      windows_enabled_and_configured: true,
    });

    expect(screen.getByText("Migration")).toBeVisible();
    expect(screen.getByRole("checkbox")).toBeVisible();
  });

  it("disables the default fleet dropdown when Fleet is not connected to Entra", () => {
    renderPage({
      windows_enabled_and_configured: true,
      windows_entra_tenant_ids: [],
    });

    expect(screen.getByText("User driven enrollment")).toBeVisible();
    expect(screen.getByText("Default fleet")).toBeVisible();
    expect(screen.getByRole("combobox")).toBeDisabled();
  });

  it("enables the default fleet dropdown when Fleet is connected to Entra", () => {
    renderPage({
      windows_enabled_and_configured: true,
      windows_entra_tenant_ids: ["tenant-1"],
    });

    expect(screen.getByRole("combobox")).toBeEnabled();
  });

  it("saves the toggle states and the default fleet through the config API", async () => {
    (configAPI.updateMDMConfig as jest.Mock).mockResolvedValue({});
    const { user } = renderPage({
      windows_enabled_and_configured: true,
      enable_turn_on_windows_mdm_manually: false,
      windows_entra_tenant_ids: ["tenant-1"],
      windows_automatic_enrollment: { default_fleet: "Workstations" },
    });

    // Turning programmatic enrollment off also forces auto migration off.
    await user.click(screen.getAllByRole("switch")[1]);
    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(configAPI.updateMDMConfig).toHaveBeenCalledWith(
      {
        windows_enabled_and_configured: true,
        enable_turn_on_windows_mdm_manually: true,
        windows_migration_enabled: false,
        windows_automatic_enrollment: { default_fleet: "Workstations" },
      },
      true
    );
  });

  it("does not re-save a stale migration setting when enrollment is manual", async () => {
    (configAPI.updateMDMConfig as jest.Mock).mockResolvedValue({});
    // Inconsistent server state (settable via the API or GitOps): migration
    // enabled while enrollment is manual, so the Migration checkbox is hidden.
    const { user } = renderPage({
      windows_enabled_and_configured: true,
      enable_turn_on_windows_mdm_manually: true,
      windows_migration_enabled: true,
      windows_entra_tenant_ids: ["tenant-1"],
    });

    expect(screen.queryByText("Migration")).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(configAPI.updateMDMConfig).toHaveBeenCalledWith(
      expect.objectContaining({ windows_migration_enabled: false }),
      true
    );
  });
});
