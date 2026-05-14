import React from "react";
import { screen, waitFor } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";
import {
  createMockAppStoreAppAndroid,
  createMockAppStoreAppIos,
  createMockSoftwarePackageIos,
} from "__mocks__/softwareMock";
import EditConfigurationModal from "./EditConfigurationModal";

const androidInstaller = createMockAppStoreAppAndroid();
const iosVppInstaller = createMockAppStoreAppIos();
const iosInHouseInstaller = createMockSoftwarePackageIos();

const ANDROID_PROPS = {
  softwareId: 123,
  teamId: 456,
  softwareInstaller: androidInstaller,
  isApplePlatform: false,
  onExit: jest.fn(),
  refetchSoftwareTitle: jest.fn(),
};

const IOS_VPP_PROPS = {
  softwareId: 789,
  teamId: 456,
  softwareInstaller: iosVppInstaller,
  isApplePlatform: true,
  onExit: jest.fn(),
  refetchSoftwareTitle: jest.fn(),
};

const IOS_IN_HOUSE_PROPS = {
  softwareId: 101,
  teamId: 456,
  softwareInstaller: iosInHouseInstaller,
  isApplePlatform: true,
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

  describe("iOS/iPadOS VPP (XML)", () => {
    it("renders modal with XML help text and description", () => {
      const render = createCustomRenderer({ withBackendMock: true });
      render(<EditConfigurationModal {...IOS_VPP_PROPS} />);

      expect(screen.getByText("Edit configuration")).toBeInTheDocument();
      expect(
        screen.getByText(/Managed app configuration, also known as App Config/i)
      ).toBeInTheDocument();
      expect(
        screen.getByText(/will be applied to future installs and updates/i)
      ).toBeInTheDocument();
    });

    it("shows App Store (VPP) installer details", () => {
      const render = createCustomRenderer({ withBackendMock: true });
      render(<EditConfigurationModal {...IOS_VPP_PROPS} />);

      expect(screen.getAllByText(iosVppInstaller.name).length).toBeGreaterThan(
        0
      );
      expect(screen.getByText(/App Store \(VPP\)/)).toBeInTheDocument();
    });

    it("shows Cancel and Save buttons", () => {
      const render = createCustomRenderer({ withBackendMock: true });
      render(<EditConfigurationModal {...IOS_VPP_PROPS} />);

      expect(
        screen.getByRole("button", { name: "Cancel" })
      ).toBeInTheDocument();
      expect(screen.getByRole("button", { name: "Save" })).toBeInTheDocument();
    });

    it("initializes with valid XML (Save enabled)", async () => {
      const render = createCustomRenderer({ withBackendMock: true });
      render(<EditConfigurationModal {...IOS_VPP_PROPS} />);

      const saveButton = screen.getByRole("button", { name: "Save" });
      await waitFor(() => {
        expect(saveButton).toBeEnabled();
      });
    });

    it("links to iOS/iPadOS learn more URL", () => {
      const render = createCustomRenderer({ withBackendMock: true });
      render(<EditConfigurationModal {...IOS_VPP_PROPS} />);

      const learnMore = screen.getByText("Learn more").closest("a");
      expect(learnMore).toHaveAttribute(
        "href",
        expect.stringContaining("ios-software-managed-configuration")
      );
    });
  });

  describe("iOS/iPadOS in-house .ipa (XML)", () => {
    it("renders modal with XML help text and description", () => {
      const render = createCustomRenderer({ withBackendMock: true });
      render(<EditConfigurationModal {...IOS_IN_HOUSE_PROPS} />);

      expect(screen.getByText("Edit configuration")).toBeInTheDocument();
      expect(
        screen.getByText(/Managed app configuration, also known as App Config/i)
      ).toBeInTheDocument();
      expect(
        screen.getByText(/will be applied to future installs and updates/i)
      ).toBeInTheDocument();
    });

    it("shows installer details with package type", () => {
      const render = createCustomRenderer({ withBackendMock: true });
      render(<EditConfigurationModal {...IOS_IN_HOUSE_PROPS} />);

      expect(
        screen.getAllByText(iosInHouseInstaller.name).length
      ).toBeGreaterThan(0);
    });

    it("shows Cancel and Save buttons", () => {
      const render = createCustomRenderer({ withBackendMock: true });
      render(<EditConfigurationModal {...IOS_IN_HOUSE_PROPS} />);

      expect(
        screen.getByRole("button", { name: "Cancel" })
      ).toBeInTheDocument();
      expect(screen.getByRole("button", { name: "Save" })).toBeInTheDocument();
    });

    it("initializes with valid XML (Save enabled)", async () => {
      const render = createCustomRenderer({ withBackendMock: true });
      render(<EditConfigurationModal {...IOS_IN_HOUSE_PROPS} />);

      const saveButton = screen.getByRole("button", { name: "Save" });
      await waitFor(() => {
        expect(saveButton).toBeEnabled();
      });
    });

    it("initializes with empty configuration and Save enabled", async () => {
      const render = createCustomRenderer({ withBackendMock: true });
      const emptyInstaller = createMockSoftwarePackageIos({
        configuration: "",
      });
      render(
        <EditConfigurationModal
          {...IOS_IN_HOUSE_PROPS}
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
      const { user } = render(
        <EditConfigurationModal {...IOS_IN_HOUSE_PROPS} />
      );

      await user.click(screen.getByRole("button", { name: "Cancel" }));

      expect(IOS_IN_HOUSE_PROPS.onExit).toHaveBeenCalled();
    });
  });
});
