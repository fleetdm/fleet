import React from "react";
import { screen, waitFor } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";
import {
  createMockAppStoreAppAndroid,
  createMockAppStoreAppIos,
} from "__mocks__/softwareMock";
import EditConfigurationModal from "./EditConfigurationModal";
import { validateJson, validateXml, getPlatformLabel } from "./helpers";

// ──────────────────────────────────────────────
// Unit tests for validation helpers
// ──────────────────────────────────────────────

describe("validateJson", () => {
  it("returns null for valid JSON", () => {
    expect(validateJson('{"key":"value"}')).toBeNull();
  });

  it("returns null for empty string", () => {
    expect(validateJson("")).toBeNull();
  });

  it("returns error message for invalid JSON", () => {
    const error = validateJson("{{ invalid");
    expect(error).toBeTruthy();
    expect(typeof error).toBe("string");
  });
});

describe("validateXml", () => {
  it("returns null for valid XML with <dict> root", () => {
    expect(
      validateXml("<dict><key>k</key><string>v</string></dict>")
    ).toBeNull();
  });

  it("returns null for empty string", () => {
    expect(validateXml("")).toBeNull();
  });

  it("returns error for malformed XML", () => {
    const error = validateXml("<dict><unclosed");
    expect(error).toBeTruthy();
  });

  it("returns error when root element is not <dict>", () => {
    const error = validateXml("<array><string>hi</string></array>");
    expect(error).toMatch(/root element must be <dict>/i);
  });

  it("returns null for multi-line XML with self-closing tags", () => {
    const xml = "<dict>\n\t<key>ForceLoginWithSSO</key>\n\t<true/>\n</dict>";
    expect(validateXml(xml)).toBeNull();
  });
});

describe("getPlatformLabel", () => {
  it("returns iOS for ios", () => {
    expect(getPlatformLabel("ios")).toBe("iOS");
  });

  it("returns iPadOS for ipados", () => {
    expect(getPlatformLabel("ipados")).toBe("iPadOS");
  });

  it("returns Android for android", () => {
    expect(getPlatformLabel("android")).toBe("Android");
  });
});

// ──────────────────────────────────────────────
// Component tests
// ──────────────────────────────────────────────

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

    it("shows the installer details widget with software name and Android label", () => {
      const render = createCustomRenderer({ withBackendMock: true });
      render(<EditConfigurationModal {...ANDROID_PROPS} />);

      expect(screen.getAllByText(androidInstaller.name).length).toBeGreaterThan(
        0
      );
      expect(screen.getByText("Android")).toBeInTheDocument();
    });

    it("initializes the configuration editor with valid JSON (Save enabled)", async () => {
      const render = createCustomRenderer({ withBackendMock: true });
      render(<EditConfigurationModal {...ANDROID_PROPS} />);

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
  });

  describe("iOS/iPadOS (XML)", () => {
    it("renders modal with XML help text and iOS label", () => {
      const render = createCustomRenderer({ withBackendMock: true });
      render(<EditConfigurationModal {...IOS_PROPS} />);

      expect(screen.getByText("Edit configuration")).toBeInTheDocument();
      expect(screen.getByText("Configuration")).toBeInTheDocument();
      expect(
        screen.getByText(/The iOS app's configuration in XML format/i)
      ).toBeInTheDocument();
      expect(screen.getByText("Learn more")).toBeInTheDocument();
      expect(screen.getByRole("button", { name: "Save" })).toBeInTheDocument();
    });

    it("shows the installer details widget with software name and iOS label", () => {
      const render = createCustomRenderer({ withBackendMock: true });
      render(<EditConfigurationModal {...IOS_PROPS} />);

      expect(screen.getAllByText(iosInstaller.name).length).toBeGreaterThan(0);
      expect(screen.getByText("iOS")).toBeInTheDocument();
    });

    it("shows iPadOS label for iPadOS platform", () => {
      const render = createCustomRenderer({ withBackendMock: true });
      const ipadosInstaller = createMockAppStoreAppIos({
        platform: "ipados",
      });
      render(
        <EditConfigurationModal
          {...IOS_PROPS}
          softwareInstaller={ipadosInstaller}
        />
      );

      expect(screen.getByText("iPadOS")).toBeInTheDocument();
      expect(
        screen.getByText(/The iPadOS app's configuration in XML format/i)
      ).toBeInTheDocument();
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

    it("calls onExit handler when modal close is triggered via Escape key", async () => {
      const render = createCustomRenderer({ withBackendMock: true });
      const { user } = render(<EditConfigurationModal {...IOS_PROPS} />);

      await user.keyboard("{Escape}");

      expect(IOS_PROPS.onExit).toHaveBeenCalled();
    });
  });
});
