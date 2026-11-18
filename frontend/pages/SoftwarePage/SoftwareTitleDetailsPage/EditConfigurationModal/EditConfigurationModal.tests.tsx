import { screen, waitFor } from "@testing-library/react";
import React from "react";

import {
  createMockAppStoreAppAndroid,
  createMockAppStoreAppIos,
  createMockSoftwarePackageIos,
} from "__mocks__/softwareMock";
import { createCustomRenderer } from "test/test-utils";

import EditConfigurationModal from "./EditConfigurationModal";

// Fire the debounced validator synchronously so tests don't wait 3s for the
// on-type validation pass.
jest.mock("use-debounce", () => ({
  useDebouncedCallback: <T extends (...args: unknown[]) => unknown>(fn: T) => {
    const callback = (...args: Parameters<T>) => fn(...args);
    callback.cancel = () => undefined;
    callback.flush = () => undefined;
    return callback;
  },
}));

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

const renderWithGitOps = (
  props: React.ComponentProps<typeof EditConfigurationModal>,
  { gitOpsModeEnabled }: { gitOpsModeEnabled: boolean }
) => {
  const render = createCustomRenderer({
    withBackendMock: true,
    context: {
      app: {
        isPremiumTier: true,
        isGlobalAdmin: true,
        config: { gitops: { gitops_mode_enabled: gitOpsModeEnabled } },
      },
    },
  });
  return render(<EditConfigurationModal {...props} />);
};

describe("EditConfigurationModal", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("disables Save in GitOps mode", async () => {
    renderWithGitOps(IOS_VPP_PROPS, { gitOpsModeEnabled: true });
    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
    });
    expect(screen.getByRole("button", { name: "Cancel" })).toBeEnabled();
  });

  it("leaves Save enabled when the software exception is on in GitOps mode", async () => {
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPremiumTier: true,
          isGlobalAdmin: true,
          config: {
            gitops: {
              gitops_mode_enabled: true,
              exceptions: { software: true },
            },
          },
        },
      },
    });
    render(<EditConfigurationModal {...IOS_VPP_PROPS} />);
    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Save" })).toBeEnabled();
    });
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
        screen.queryByText(/applied during future installs/i)
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

    it("surfaces the parse error and keeps Save enabled when configuration JSON is invalid", async () => {
      const render = createCustomRenderer({ withBackendMock: true });
      const { user } = render(<EditConfigurationModal {...ANDROID_PROPS} />);

      const configInput = screen.getByRole<HTMLTextAreaElement>("textbox", {
        name: /Cursor at row/,
      });
      const saveButton = screen.getByRole("button", { name: "Save" });

      await user.clear(configInput);
      await user.type(configInput, "{{ invalid json");

      await waitFor(() => {
        expect(
          screen.getByText(/Expected property name or '}'/i)
        ).toBeInTheDocument();
      });
      expect(saveButton).toBeEnabled();
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
        screen.getByText(/applied during future installs and updates/i)
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
        screen.getByText(/applied during future installs and updates/i)
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
