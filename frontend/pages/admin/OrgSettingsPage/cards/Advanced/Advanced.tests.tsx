import React from "react";
import { screen } from "@testing-library/react";
import { renderWithSetup, createMockRouter } from "test/test-utils";

import createMockConfig from "__mocks__/configMock";

import Advanced from "./Advanced";

const renderAdvanced = (
  overrides: {
    historicalData?: { uptime: boolean; vulnerabilities: boolean };
    handleSubmit?: jest.Mock;
  } = {}
) => {
  const baseConfig = createMockConfig();
  const config = {
    ...baseConfig,
    features: {
      ...baseConfig.features,
      historical_data:
        overrides.historicalData ?? baseConfig.features.historical_data,
    },
  };
  const handleSubmit =
    overrides.handleSubmit ?? jest.fn().mockResolvedValue(true);
  const utils = renderWithSetup(
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
      screen.getByLabelText(/Disable hosts online historical reporting/i)
    ).toBeInTheDocument();
    expect(
      screen.getByLabelText(
        /Disable vulnerability exposure historical reporting/i
      )
    ).toBeInTheDocument();
  });

  it("starts with both checkboxes unchecked when collection is enabled", () => {
    renderAdvanced();
    expect(
      screen.getByLabelText(/Disable hosts online historical reporting/i)
    ).not.toBeChecked();
    expect(
      screen.getByLabelText(/Disable vulnerability exposure historical reporting/i)
    ).not.toBeChecked();
  });

  it("starts with the checkbox checked when collection is disabled in config", () => {
    renderAdvanced({
      historicalData: { uptime: false, vulnerabilities: true },
    });
    expect(screen.getByLabelText(/Disable hosts online/i)).toBeChecked();
    expect(
      screen.getByLabelText(/Disable vulnerability exposure historical reporting/i)
    ).not.toBeChecked();
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
