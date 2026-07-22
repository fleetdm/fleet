import React from "react";

import { screen } from "@testing-library/react";

import { createMockRouter, createCustomRenderer } from "test/test-utils";
import { createMockConfig } from "__mocks__/configMock";
import mdmAndroidAPI from "services/entities/mdm_android";

import AndroidMdmPage from "./AndroidMdmPage";

const createGitOpsConfig = () =>
  createMockConfig({
    gitops: {
      gitops_mode_enabled: true,
      repository_url: "https://example.com/repo",
      exceptions: { labels: false, software: false, secrets: true },
    },
  });

describe("AndroidMdmPage", () => {
  afterEach(() => {
    jest.restoreAllMocks();
  });

  it("disables the Connect button in GitOps mode", () => {
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isAndroidMdmEnabledAndConfigured: false,
          config: createGitOpsConfig(),
        },
      },
    });

    render(<AndroidMdmPage router={createMockRouter()} />);

    const connectButton = screen.getByRole("button", { name: "Connect" });
    expect(connectButton).toBeDisabled();
    // The button is disabled by the GitOps wrapper (which only renders its span
    // in GitOps mode), not by some unrelated state.
    expect(
      connectButton.closest(".gitops-mode-tooltip-wrapper")
    ).toBeInTheDocument();
  });

  it("enables the Connect button when not in GitOps mode", () => {
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isAndroidMdmEnabledAndConfigured: false,
          config: createMockConfig(),
        },
      },
    });

    render(<AndroidMdmPage router={createMockRouter()} />);

    const connectButton = screen.getByRole("button", { name: "Connect" });
    expect(connectButton).toBeEnabled();
    expect(
      connectButton.closest(".gitops-mode-tooltip-wrapper")
    ).not.toBeInTheDocument();
  });

  it("disables the Turn off Android MDM button in GitOps mode", async () => {
    jest
      .spyOn(mdmAndroidAPI, "getAndroidEnterprise")
      .mockResolvedValue({ android_enterprise_id: true });

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isAndroidMdmEnabledAndConfigured: true,
          config: createGitOpsConfig(),
        },
      },
    });

    render(<AndroidMdmPage router={createMockRouter()} />);

    const turnOffButton = await screen.findByRole("button", {
      name: "Turn off Android MDM",
    });
    expect(turnOffButton).toBeDisabled();
    expect(
      turnOffButton.closest(".gitops-mode-tooltip-wrapper")
    ).toBeInTheDocument();
  });
});
