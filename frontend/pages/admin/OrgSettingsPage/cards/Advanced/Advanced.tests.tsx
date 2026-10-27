import { screen } from "@testing-library/react";
import React from "react";

import createMockConfig from "__mocks__/configMock";
import { createCustomRenderer, createMockRouter } from "test/test-utils";

import Advanced from "./Advanced";

const renderAdvanced = (
  overrides: {
    historicalData?: { uptime: boolean; vulnerabilities: boolean };
    handleSubmit?: jest.Mock;
    tier?: string;
    mcpDisabled?: boolean;
  } = {}
) => {
  const baseConfig = createMockConfig();
  const config = {
    ...baseConfig,
    license: { ...baseConfig.license, tier: overrides.tier ?? "premium" },
    server_settings: {
      ...baseConfig.server_settings,
      mcp_disabled: overrides.mcpDisabled ?? false,
    },
    features: {
      ...baseConfig.features,
      historical_data:
        overrides.historicalData ?? baseConfig.features.historical_data,
    },
  };
  const handleSubmit =
    overrides.handleSubmit ?? jest.fn().mockResolvedValue(true);
  const render = createCustomRenderer({
    context: { app: { isPremiumTier: true } },
  });
  const utils = render(
    <Advanced
      appConfig={config}
      handleSubmit={handleSubmit}
      isUpdatingSettings={false}
      router={createMockRouter()}
    />
  );
  return { ...utils, handleSubmit, config };
};

describe("Advanced settings — Activity & data retention", () => {
  it("renders the new section heading and both checkboxes", () => {
    renderAdvanced();
    expect(screen.getByText("Activity & data retention")).toBeInTheDocument();
    expect(
      screen.getByLabelText(/Hosts online historical reporting/i)
    ).toBeInTheDocument();
    expect(
      screen.getByLabelText(/Vulnerability exposure historical reporting/i)
    ).toBeInTheDocument();
  });

  it("starts with both checkboxes checked when collection is enabled", () => {
    renderAdvanced();
    expect(
      screen.getByLabelText(/Hosts online historical reporting/i)
    ).toBeChecked();
    expect(
      screen.getByLabelText(/Vulnerability exposure historical reporting/i)
    ).toBeChecked();
  });

  it("starts with the checkbox unchecked when collection is disabled in config", () => {
    renderAdvanced({
      historicalData: { uptime: false, vulnerabilities: true },
    });
    expect(
      screen.getByLabelText(/Hosts online historical reporting/i)
    ).not.toBeChecked();
    expect(
      screen.getByLabelText(/Vulnerability exposure historical reporting/i)
    ).toBeChecked();
  });

  it("submits without confirmation when no dataset is being newly disabled", async () => {
    const { user, handleSubmit } = renderAdvanced();
    await user.click(screen.getByRole("button", { name: /^save$/i }));
    expect(handleSubmit).toHaveBeenCalledTimes(1);
    const payload = handleSubmit.mock.calls[0][0];
    expect(payload.features.historical_data).toEqual({
      uptime: true,
      vulnerabilities: true,
    });
    // Modal should not have appeared
    expect(
      screen.queryByText("Disable data collection")
    ).not.toBeInTheDocument();
  });

  it("opens the confirmation modal when a dataset is being disabled", async () => {
    const { user, handleSubmit } = renderAdvanced();
    await user.click(
      screen.getByRole("checkbox", { name: "disableHostsActive" })
    );
    await user.click(screen.getByRole("button", { name: /^save$/i }));
    expect(handleSubmit).not.toHaveBeenCalled();
    expect(screen.getByText("Disable data collection")).toBeInTheDocument();
  });

  it("issues the PATCH after the user confirms via the modal", async () => {
    const { user, handleSubmit } = renderAdvanced();
    await user.click(
      screen.getByRole("checkbox", { name: "disableHostsActive" })
    );
    await user.click(screen.getByRole("button", { name: /^save$/i }));
    await user.click(screen.getByRole("button", { name: /save and disable/i }));
    expect(handleSubmit).toHaveBeenCalledTimes(1);
    const payload = handleSubmit.mock.calls[0][0];
    expect(payload.features.historical_data).toEqual({
      uptime: false,
      vulnerabilities: true,
    });
  });

  it("does not open the modal when re-enabling a previously-disabled dataset", async () => {
    const { user, handleSubmit } = renderAdvanced({
      historicalData: { uptime: false, vulnerabilities: true },
    });
    await user.click(
      screen.getByRole("checkbox", { name: "disableHostsActive" })
    );
    await user.click(screen.getByRole("button", { name: /^save$/i }));
    expect(handleSubmit).toHaveBeenCalledTimes(1);
    expect(
      screen.queryByText("Disable data collection")
    ).not.toBeInTheDocument();
    const payload = handleSubmit.mock.calls[0][0];
    expect(payload.features.historical_data.uptime).toBe(true);
  });
});

describe("Advanced settings — Host lifecycle", () => {
  it("disables the host expiry window until host expiry is enabled", async () => {
    const { user } = renderAdvanced();

    expect(screen.getByLabelText("Host expiry window")).toBeDisabled();

    await user.click(screen.getByText("Host expiry"));

    expect(screen.getByLabelText("Host expiry window")).toBeEnabled();
  });
});

describe("Advanced settings — MCP server", () => {
  it("shows the MCP server checkbox checked by default on Premium", () => {
    renderAdvanced();
    expect(screen.getByLabelText(/MCP server/)).toBeChecked();
  });

  it("links the MCP server help text to the guide", () => {
    renderAdvanced();
    expect(screen.getByText(/ask questions about your fleet/)).toBeVisible();
    const guideLinks = screen
      .getAllByRole("link", { name: /learn more/i })
      .filter(
        (link) =>
          link.getAttribute("href") === "https://fleetdm.com/guides/fleet-mcp"
      );
    expect(guideLinks).toHaveLength(1);
  });

  it("sends mcp_disabled true after unchecking it", async () => {
    const { user, handleSubmit } = renderAdvanced();
    await user.click(screen.getByRole("checkbox", { name: "disableMCP" }));
    await user.click(screen.getByRole("button", { name: /^save$/i }));
    expect(handleSubmit.mock.calls[0][0].server_settings.mcp_disabled).toBe(
      true
    );
  });

  it("starts unchecked when the MCP server is disabled in config", () => {
    renderAdvanced({ mcpDisabled: true });
    expect(screen.getByLabelText(/MCP server/)).not.toBeChecked();
  });

  it("hides the MCP server checkbox on Fleet Free", () => {
    renderAdvanced({ tier: "free" });
    expect(screen.queryByLabelText(/MCP server/)).not.toBeInTheDocument();
  });
});
