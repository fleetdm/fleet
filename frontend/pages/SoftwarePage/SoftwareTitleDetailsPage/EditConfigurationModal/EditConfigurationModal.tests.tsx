import React from "react";
import { screen, waitFor } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";
import { createMockAppStoreAppAndroid } from "__mocks__/softwareMock";
import softwareAPI from "services/entities/software";
import EditConfigurationModal from "./EditConfigurationModal";

const softwareInstaller = createMockAppStoreAppAndroid();

const MOCK_PROPS = {
  softwareId: 123,
  teamId: 456,
  softwareInstaller,
  onExit: jest.fn(),
  refetchSoftwareTitle: jest.fn(),
};

describe("EditConfigurationModal", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("renders modal title, configuration editor, help text, and save button", () => {
    const render = createCustomRenderer({ withBackendMock: true });
    render(<EditConfigurationModal {...MOCK_PROPS} />);

    expect(screen.getByText("Edit configuration")).toBeInTheDocument();

    // Editor label
    expect(screen.getByText("Configuration")).toBeInTheDocument();

    // Help text / learn more link
    expect(
      screen.getByText(/The Android app's configuration in JSON format/i)
    ).toBeInTheDocument();
    expect(screen.getByText("Learn more")).toBeInTheDocument();

    const save = screen.getByRole("button", { name: "Save" });
    expect(save).toBeInTheDocument();
  });

  it("shows the installer details widget with software name", () => {
    const render = createCustomRenderer({ withBackendMock: true });
    render(<EditConfigurationModal {...MOCK_PROPS} />);

    // InstallerDetailsWidget should show the software name somewhere
    expect(screen.getAllByText(softwareInstaller.name).length).toBeGreaterThan(
      0
    );

    // CustomDetails is "Android" in your props
    expect(screen.getByText("Android")).toBeInTheDocument();
  });

  it("initializes the configuration editor with valid JSON (Save enabled)", async () => {
    const render = createCustomRenderer({ withBackendMock: true });
    render(<EditConfigurationModal {...MOCK_PROPS} />);

    const saveButton = screen.getByRole("button", { name: "Save" });

    await waitFor(() => {
      expect(saveButton).toBeEnabled();
    });
  });

  it("calls onExit handler when modal close is triggered via Escape key", async () => {
    const render = createCustomRenderer({ withBackendMock: true });
    const { user } = render(<EditConfigurationModal {...MOCK_PROPS} />);

    await user.keyboard("{Escape}");

    expect(MOCK_PROPS.onExit).toHaveBeenCalled();
  });

  it("disables Save button when configuration JSON is invalid and shows the error", async () => {
    const render = createCustomRenderer({ withBackendMock: true });
    const { user } = render(<EditConfigurationModal {...MOCK_PROPS} />);

    const configInput = screen.getByRole<HTMLTextAreaElement>("textbox", {
      name: "",
    });
    const saveButton = screen.getByRole("button", { name: "Save" });

    // Type some invalid JSON
    await user.clear(configInput);
    await user.type(configInput, "{{ invalid json");

    // Error is rendered and Save disabled
    await waitFor(() => {
      expect(saveButton).toBeDisabled();
    });

    expect(
      screen.getByText(/Expected property name or '}'/i)
    ).toBeInTheDocument();
  });

  it("enables Save button when an empty object when configuration field is cleared", async () => {
    const render = createCustomRenderer({ withBackendMock: true });
    const { user } = render(<EditConfigurationModal {...MOCK_PROPS} />);

    const configInput = screen.getByRole<HTMLTextAreaElement>("textbox", {
      name: "",
    });
    const saveButton = screen.getByRole("button", { name: "Save" });

    await user.clear(configInput);

    await waitFor(() => {
      expect(saveButton).not.toBeDisabled();
    });
  });
});
