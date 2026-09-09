import React from "react";
import { screen } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";

import createMockConfig from "__mocks__/configMock";

import HostActivityAutomationsModal from "./HostActivityAutomationsModal";

const defaultProps = {
  fleetName: "Workstations",
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
    // Description names the selected fleet.
    expect(screen.getByText("Workstations")).toBeInTheDocument();
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

  it("disables Save while an update is in flight", () => {
    const render = createCustomRenderer({
      context: { app: { config: createMockConfig() } },
    });

    render(
      <HostActivityAutomationsModal
        {...defaultProps}
        isUpdating
        automationSettings={{
          enable_host_activities_webhook: true,
          destination_url: "https://example.com/hook",
        }}
      />
    );

    expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
  });

  it("clears an invalid URL and its error when disabling, keeps it cleared on re-enable", async () => {
    const render = createCustomRenderer({
      context: { app: { config: createMockConfig() } },
    });

    const { user } = render(
      <HostActivityAutomationsModal
        {...defaultProps}
        automationSettings={{
          enable_host_activities_webhook: true,
          destination_url: "not-a-url",
        }}
      />
    );

    // Surface the validation error, then disable the webhook.
    await user.click(screen.getByRole("button", { name: "Save" }));
    expect(
      screen.getByText("not-a-url is not a valid destination URL")
    ).toBeInTheDocument();

    await user.click(screen.getByRole("switch"));
    expect(
      screen.queryByText("not-a-url is not a valid destination URL")
    ).not.toBeInTheDocument();

    // Re-enabling starts clean: the invalid URL was dropped.
    await user.click(screen.getByRole("switch"));
    expect(screen.queryByDisplayValue("not-a-url")).not.toBeInTheDocument();
  });

  it("keeps a valid URL when the webhook is toggled off and back on", async () => {
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

    await user.click(screen.getByRole("switch"));
    await user.click(screen.getByRole("switch"));
    expect(
      screen.getByDisplayValue("https://example.com/hook")
    ).toBeInTheDocument();
  });

  it("blocks saving in GitOps mode", async () => {
    const onSubmit = jest.fn();
    const config = createMockConfig();
    config.gitops = { ...config.gitops, gitops_mode_enabled: true };
    const render = createCustomRenderer({
      context: { app: { config } },
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

    expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();

    // The Modal's Enter handler must not submit either.
    await user.keyboard("{Enter}");
    expect(onSubmit).not.toHaveBeenCalled();
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
