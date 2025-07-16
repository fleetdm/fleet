import React from "react";
import { screen } from "@testing-library/react";

import { createMockConfig, createMockMdmConfig } from "__mocks__/configMock";
import { IConfig } from "interfaces/config";
import { createCustomRenderer, createMockRouter } from "test/test-utils";

import EndUserMigrationSection from "./EndUserMigrationSection";

const createTestMockData = (
  configOverrides: Partial<IConfig>,
  isPremiumTier = true
) => {
  return {
    context: {
      app: {
        isPremiumTier,
        config: createMockConfig({
          ...configOverrides,
        }),
        setConfig: jest.fn(),
      },
      notification: {
        renderFlash: jest.fn(),
      },
    },
  };
};

describe("EndUserMigrationSection", () => {
  const mockRouter = createMockRouter();

  it("toggles form elements disabled state when slider is clicked", async () => {
    const render = createCustomRenderer(
      createTestMockData({
        mdm: createMockMdmConfig({
          macos_migration: {
            enable: false,
            mode: "voluntary",
            webhook_url: "",
          },
        }),
      })
    );

    const { user } = render(<EndUserMigrationSection router={mockRouter} />);

    // Verify slider is initially disabled (off)
    const slider = screen.getByRole("switch");
    expect(slider).not.toBeChecked();

    // Verify form elements are disabled
    const voluntaryRadio = screen.getByRole("radio", { name: "Voluntary" });
    const forcedRadio = screen.getByRole("radio", { name: "Forced" });
    const webhookInput = screen.getByRole("textbox", { name: "Webhook URL" });
    expect(voluntaryRadio).toBeDisabled();
    expect(forcedRadio).toBeDisabled();
    expect(webhookInput).toBeDisabled();

    // Click the slider to enable it form elements.
    // have to wait for the async state update
    user.click(slider);
    await screen.findByRole("switch", { checked: true });

    expect(slider).toBeChecked();
    expect(voluntaryRadio).not.toBeDisabled();
    expect(forcedRadio).not.toBeDisabled();
    expect(webhookInput).not.toBeDisabled();
  });

  it("disables form elements when gitops mode is enabled", async () => {
    const render = createCustomRenderer(
      createTestMockData({
        mdm: createMockMdmConfig({
          macos_migration: {
            enable: true,
            mode: "voluntary",
            webhook_url: "",
          },
        }),
        gitops: {
          gitops_mode_enabled: true,
          repository_url: "https://example.com/repo.git",
        },
      })
    );

    const { user } = render(<EndUserMigrationSection router={mockRouter} />);

    // Verify slider is enabled but disabled due to gitops mode
    const slider = screen.getByRole("switch");
    expect(slider).toBeChecked();
    expect(slider).toBeDisabled();

    // Verify form elements are disabled
    const voluntaryRadio = screen.getByRole("radio", { name: "Voluntary" });
    const forcedRadio = screen.getByRole("radio", { name: "Forced" });
    const webhookInput = screen.getByRole("textbox", { name: "Webhook URL" });

    expect(voluntaryRadio).toBeDisabled();
    expect(forcedRadio).toBeDisabled();
    expect(webhookInput).toBeDisabled();

    // clicking the slider should have no effect
    user.click(slider);
    expect(slider).toBeDisabled();
    expect(voluntaryRadio).toBeDisabled();
    expect(forcedRadio).toBeDisabled();
    expect(webhookInput).toBeDisabled();
  });

  it("renders the connect button when MDM is not connected", () => {
    const render = createCustomRenderer(
      createTestMockData({
        mdm: createMockMdmConfig({
          apple_bm_enabled_and_configured: false,
        }),
      })
    );

    render(<EndUserMigrationSection router={mockRouter} />);

    expect(
      screen.getByText("Connect to Apple Business Manager to get started.")
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Connect" })).toBeInTheDocument();
  });

  it("renders the premium feature message when not on premium tier", () => {
    const render = createCustomRenderer(createTestMockData({}, false));

    render(<EndUserMigrationSection router={mockRouter} />);

    expect(
      screen.getByText("This feature is included in Fleet Premium.")
    ).toBeInTheDocument();
  });
});
