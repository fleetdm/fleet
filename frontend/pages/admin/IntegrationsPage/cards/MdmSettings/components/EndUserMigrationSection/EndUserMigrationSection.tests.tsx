import React from "react";
import { screen, waitFor } from "@testing-library/react";

import { createMockConfig, createMockMdmConfig } from "__mocks__/configMock";
import { IConfig } from "interfaces/config";
import configAPI from "services/entities/config";
import { createCustomRenderer, createMockRouter } from "test/test-utils";

import EndUserMigrationSection from "./EndUserMigrationSection";

jest.mock("services/entities/config");
jest.mock("components/ToastNotification", () => ({
  notify: {
    success: jest.fn(),
    error: jest.fn(),
    batch: jest.fn(),
    dismiss: jest.fn(),
  },
}));

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
    },
  };
};

describe("EndUserMigrationSection", () => {
  const mockRouter = createMockRouter();

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("disables the save button while an update is in flight", async () => {
    // Hold the request open so the in-flight window can be asserted on.
    let resolveUpdate!: (config: IConfig) => void;
    const updateSpy = jest.mocked(configAPI.update).mockImplementation(
      () =>
        new Promise<IConfig>((resolve) => {
          resolveUpdate = resolve;
        })
    );

    const render = createCustomRenderer(
      createTestMockData({
        mdm: createMockMdmConfig({
          macos_migration: {
            enable: true,
            mode: "voluntary",
            webhook_url: "https://example.com/webhook",
          },
        }),
      })
    );

    const { user } = render(<EndUserMigrationSection router={mockRouter} />);

    const saveButton = screen.getByRole("button", { name: "Save" });
    expect(saveButton).not.toBeDisabled();

    await user.click(saveButton);
    await waitFor(() => expect(saveButton).toBeDisabled());

    // Extra clicks while the first request is open must not send more requests.
    await user.click(saveButton);
    await user.click(saveButton);
    expect(updateSpy).toHaveBeenCalledTimes(1);

    resolveUpdate(createMockConfig());
    await waitFor(() => expect(saveButton).not.toBeDisabled());
  });

  it("re-enables the save button when the update fails", async () => {
    // Hold the request open so the button can be observed disabled before the
    // failure, proving it is the rejection that re-enables it.
    let rejectUpdate!: (err: Error) => void;
    jest.mocked(configAPI.update).mockImplementation(
      () =>
        new Promise<IConfig>((_resolve, reject) => {
          rejectUpdate = reject;
        })
    );

    const render = createCustomRenderer(
      createTestMockData({
        mdm: createMockMdmConfig({
          macos_migration: {
            enable: true,
            mode: "voluntary",
            webhook_url: "https://example.com/webhook",
          },
        }),
      })
    );

    const { user } = render(<EndUserMigrationSection router={mockRouter} />);

    const saveButton = screen.getByRole("button", { name: "Save" });
    await user.click(saveButton);
    await waitFor(() => expect(saveButton).toBeDisabled());

    rejectUpdate(new Error("Something went wrong"));
    await waitFor(() => expect(saveButton).not.toBeDisabled());
  });

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
          exceptions: { labels: false, software: false, secrets: true },
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
      screen.getByText("Connect to Apple Business to get started.")
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
