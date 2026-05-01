import React from "react";
import { screen, waitFor } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";
import {
  createMockAppStoreAppAndroid,
  createMockAppStoreAppIos,
} from "__mocks__/softwareMock";
import EditConfigurationModal from "./EditConfigurationModal";

const androidInstaller = createMockAppStoreAppAndroid();
const iosInstaller = createMockAppStoreAppIos();

const ANDROID_PROPS = {
  softwareId: 123,
  teamId: 456,
  softwareInstaller: androidInstaller,
  onExit: jest.fn(),
  refetchSoftwareTitle: jest.fn(),
};

const IOS_PROPS = {
  softwareId: 789,
  teamId: 456,
  softwareInstaller: iosInstaller,
  onExit: jest.fn(),
  refetchSoftwareTitle: jest.fn(),
};

describe("EditConfigurationModal", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe("Android (JSON)", () => {
    it("renders modal title, configuration editor, help text, and save button", () => {
      const render = createCustomRenderer({ withBackendMock: true });
      render(<EditConfigurationModal {...ANDROID_PROPS} />);

      expect(screen.getByText("Edit configuration")).toBeInTheDocument();
      expect(screen.getByText("Configuration")).toBeInTheDocument();
      expect(
        screen.getByText(/The Android app's configuration in JSON format/i)
      ).toBeInTheDocument();
      expect(screen.getByText("Learn more")).toBeInTheDocument();
      expect(screen.getByRole("button", { name: "Save" })).toBeInTheDocument();
    });

    it("shows the installer details widget with Android label", () => {
      const render = createCustomRenderer({ withBackendMock: true });
      render(<EditConfigurationModal {...ANDROID_PROPS} />);

      expect(screen.getAllByText(androidInstaller.name).length).toBeGreaterThan(
        0
      );
      expect(screen.getByText("Android")).toBeInTheDocument();
    });

    it("does not render description text or Cancel button", () => {
      const render = createCustomRenderer({ withBackendMock: true });
      render(<EditConfigurationModal {...ANDROID_PROPS} />);

      expect(
        screen.queryByText(/will be applied to future installs/i)
      ).not.toBeInTheDocument();
      expect(
        screen.queryByRole("button", { name: "Cancel" })
      ).not.toBeInTheDocument();
    });

    it("initializes the configuration editor with valid JSON (Save enabled)", async () => {
      const render = createCustomRenderer({ withBackendMock: true });
      render(<EditConfigurationModal {...ANDROID_PROPS} />);

      const saveButton = screen.getByRole("button", { name: "Save" });

      await waitFor(() => {
        expect(saveButton).toBeEnabled();
      });
    });

    it("initializes with empty configuration and Save enabled", async () => {
      const render = createCustomRenderer({ withBackendMock: true });
      const emptyInstaller = createMockAppStoreAppAndroid({
        configuration: undefined,
      });
      render(
        <EditConfigurationModal
          {...ANDROID_PROPS}
          softwareInstaller={emptyInstaller}
        />
      );

      const saveButton = screen.getByRole("button", { name: "Save" });

      await waitFor(() => {
        expect(saveButton).toBeEnabled();
      });
    });

    it("calls onExit handler when modal close is triggered via Escape key", async () => {
      const render = createCustomRenderer({ withBackendMock: true });
      const { user } = render(<EditConfigurationModal {...ANDROID_PROPS} />);

      await user.keyboard("{Escape}");

      expect(ANDROID_PROPS.onExit).toHaveBeenCalled();
    });

    it("disables Save button when configuration JSON is invalid and shows the error", async () => {
      const render = createCustomRenderer({ withBackendMock: true });
      const { user } = render(<EditConfigurationModal {...ANDROID_PROPS} />);

      const configInput = screen.getByRole<HTMLTextAreaElement>("textbox", {
        name: /Cursor at row/,
      });
      const saveButton = screen.getByRole("button", { name: "Save" });

      await user.clear(configInput);
      await user.type(configInput, "{{ invalid json");

      await waitFor(() => {
        expect(saveButton).toBeDisabled();
      });

      expect(
        screen.getByText(/Expected property name or '}'/i)
      ).toBeInTheDocument();
    });

    it("enables Save button when configuration field is cleared", async () => {
      const render = createCustomRenderer({ withBackendMock: true });
      const { user } = render(<EditConfigurationModal {...ANDROID_PROPS} />);

      const configInput = screen.getByRole<HTMLTextAreaElement>("textbox", {
        name: /Cursor at row/,
      });
      const saveButton = screen.getByRole("button", { name: "Save" });

      await user.clear(configInput);

      await waitFor(() => {
        expect(saveButton).not.toBeDisabled();
      });
    });

    it("links to Android learn more URL", () => {
      const render = createCustomRenderer({ withBackendMock: true });
      render(<EditConfigurationModal {...ANDROID_PROPS} />);

      const learnMore = screen.getByText("Learn more").closest("a");
      expect(learnMore).toHaveAttribute(
        "href",
        expect.stringContaining("android-software-managed-configuration")
      );
    });
  });

  describe("iOS/iPadOS (XML)", () => {
    it("renders modal with Figma-matching help text", () => {
      const render = createCustomRenderer({ withBackendMock: true });
      render(<EditConfigurationModal {...IOS_PROPS} />);

      expect(screen.getByText("Edit configuration")).toBeInTheDocument();
      expect(screen.getByText("Configuration")).toBeInTheDocument();
      expect(
        screen.getByText(/Managed app configuration, also known as App Config/i)
      ).toBeInTheDocument();
      expect(screen.getByRole("button", { name: "Save" })).toBeInTheDocument();
    });

    it("shows description text with variables link", () => {
      const render = createCustomRenderer({ withBackendMock: true });
      render(<EditConfigurationModal {...IOS_PROPS} />);

      expect(
        screen.getByText(/will be applied to future installs and updates/i)
      ).toBeInTheDocument();
      expect(screen.getByText("variables")).toBeInTheDocument();
    });

    it("shows Cancel and Save buttons", () => {
      const render = createCustomRenderer({ withBackendMock: true });
      render(<EditConfigurationModal {...IOS_PROPS} />);

      expect(
        screen.getByRole("button", { name: "Cancel" })
      ).toBeInTheDocument();
      expect(screen.getByRole("button", { name: "Save" })).toBeInTheDocument();
    });

    it("shows App Store (VPP) installer details (not custom platform label)", () => {
      const render = createCustomRenderer({ withBackendMock: true });
      render(<EditConfigurationModal {...IOS_PROPS} />);

      expect(screen.getAllByText(iosInstaller.name).length).toBeGreaterThan(0);
      // Standard rendering shows "App Store (VPP)" not a custom "iOS" label
      expect(screen.getByText(/App Store \(VPP\)/)).toBeInTheDocument();
    });

    it("initializes with valid XML (Save enabled)", async () => {
      const render = createCustomRenderer({ withBackendMock: true });
      render(<EditConfigurationModal {...IOS_PROPS} />);

      const saveButton = screen.getByRole("button", { name: "Save" });

      await waitFor(() => {
        expect(saveButton).toBeEnabled();
      });
    });

    it("disables Save button when configuration XML is invalid", async () => {
      const render = createCustomRenderer({ withBackendMock: true });
      const { user } = render(<EditConfigurationModal {...IOS_PROPS} />);

      const configInput = screen.getByRole<HTMLTextAreaElement>("textbox", {
        name: /Cursor at row/,
      });
      const saveButton = screen.getByRole("button", { name: "Save" });

      await user.clear(configInput);
      await user.type(configInput, "<dict><unclosed");

      await waitFor(() => {
        expect(saveButton).toBeDisabled();
      });
    });

    it("initializes with empty configuration and Save enabled", async () => {
      const render = createCustomRenderer({ withBackendMock: true });
      const emptyInstaller = createMockAppStoreAppIos({
        configuration: "",
      });
      render(
        <EditConfigurationModal
          {...IOS_PROPS}
          softwareInstaller={emptyInstaller}
        />
      );

      const saveButton = screen.getByRole("button", { name: "Save" });

      await waitFor(() => {
        expect(saveButton).toBeEnabled();
      });
    });

    it("calls onExit when Cancel is clicked", async () => {
      const render = createCustomRenderer({ withBackendMock: true });
      const { user } = render(<EditConfigurationModal {...IOS_PROPS} />);

      await user.click(screen.getByRole("button", { name: "Cancel" }));

      expect(IOS_PROPS.onExit).toHaveBeenCalled();
    });

    it("links to iOS/iPadOS learn more URL", () => {
      const render = createCustomRenderer({ withBackendMock: true });
      render(<EditConfigurationModal {...IOS_PROPS} />);

      const learnMore = screen.getByText("Learn more").closest("a");
      expect(learnMore).toHaveAttribute(
        "href",
        expect.stringContaining("ios-ipados-software-managed-configuration")
      );
    });
  });
});
