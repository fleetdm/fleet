import React from "react";
import { screen } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";

import createMockConfig from "__mocks__/configMock";

import HostActivityAutomationsModal from "./HostActivityAutomationsModal";

const defaultProps = {
  onSubmit: jest.fn(),
  onExit: jest.fn(),
  isUpdating: false,
};

describe("HostActivityAutomationsModal", () => {
  it("renders stored settings when the webhook is configured", () => {
    const render = createCustomRenderer({
      context: { app: { config: createMockConfig() } },
    });

    render(
      <HostActivityAutomationsModal
        {...defaultProps}
        automationSettings={{
          enable_host_activities_webhook: true,
          destination_url: "https://example.com/hook",
        }}
      />
    );

    expect(screen.getByText("Enabled")).toBeInTheDocument();
    expect(
      screen.getByDisplayValue("https://example.com/hook")
    ).toBeInTheDocument();
  });

  it("renders disabled defaults when the webhook was never configured", () => {
    const render = createCustomRenderer({
      context: { app: { config: createMockConfig() } },
    });

    render(
      <HostActivityAutomationsModal
        {...defaultProps}
        automationSettings={null}
      />
    );

    expect(screen.getByText("Disabled")).toBeInTheDocument();
  });

  it("blocks submit with an inline error when enabled without a valid URL", async () => {
    const onSubmit = jest.fn();
    const render = createCustomRenderer({
      context: { app: { config: createMockConfig() } },
    });

    const { user } = render(
      <HostActivityAutomationsModal
        {...defaultProps}
        onSubmit={onSubmit}
        automationSettings={{
          enable_host_activities_webhook: true,
          destination_url: "",
        }}
      />
    );

    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(
      screen.getByText("Please enter a valid destination URL")
    ).toBeInTheDocument();
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it("submits the form data when valid", async () => {
    const onSubmit = jest.fn();
    const render = createCustomRenderer({
      context: { app: { config: createMockConfig() } },
    });

    const { user } = render(
      <HostActivityAutomationsModal
        {...defaultProps}
        onSubmit={onSubmit}
        automationSettings={{
          enable_host_activities_webhook: true,
          destination_url: "https://example.com/hook",
        }}
      />
    );

    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(onSubmit).toHaveBeenCalledWith({
      enabled: true,
      url: "https://example.com/hook",
    });
  });

  it("shows an example payload for a host activity", async () => {
    const render = createCustomRenderer({
      context: { app: { config: createMockConfig() } },
    });

    const { user } = render(
      <HostActivityAutomationsModal
        {...defaultProps}
        automationSettings={{
          enable_host_activities_webhook: true,
          destination_url: "https://example.com/hook",
        }}
      />
    );

    await user.click(screen.getByRole("button", { name: /Example payload/ }));

    // Host activity example: identifies the host, and the envelope has no
    // activity_id field (the backend does not send one).
    expect(screen.getByText(/"host_id"/)).toBeInTheDocument();
    expect(screen.queryByText(/"activity_id"/)).not.toBeInTheDocument();
  });

  it("disables inputs in GitOps mode", () => {
    const config = createMockConfig();
    config.gitops = { ...config.gitops, gitops_mode_enabled: true };
    const render = createCustomRenderer({
      context: { app: { config } },
    });

    render(
      <HostActivityAutomationsModal
        {...defaultProps}
        automationSettings={{
          enable_host_activities_webhook: true,
          destination_url: "https://example.com/hook",
        }}
      />
    );

    expect(screen.getByRole("switch")).toBeDisabled();
  });
});
