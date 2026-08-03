import React from "react";
import { screen, waitFor } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";
import {
  createMockSoftwarePackage,
  createMockSoftwareTitle,
} from "__mocks__/softwareMock";
import { InstallerType } from "interfaces/software";
import softwareAPI from "services/entities/software";
import EditIconModal from "./EditIconModal";

const software = createMockSoftwareTitle();
const softwarePackage = createMockSoftwarePackage();
const MOCK_PROPS = {
  softwareId: 123,
  teamIdForApi: 456,
  software: softwarePackage,
  onExit: jest.fn(),
  refetchSoftwareTitle: jest.fn(),
  iconUploadedAt: "2025-09-03T12:00:00Z",
  setIconUploadedAt: jest.fn(),
  installerType: "package" as InstallerType,
  previewInfo: {
    type: "apps",
    versions: software.versions?.length,
    source: software.source,
    currentIconUrl: null,
    name: software.name,
    titleName: software.name,
    countsUpdatedAt: "2025-09-03T12:00:00Z",
  },
};

describe("EditIconModal", () => {
  it("renders with the correct modal title for software, FileUploader, Preview tabs, save button", () => {
    const render = createCustomRenderer({ withBackendMock: true });
    render(<EditIconModal {...MOCK_PROPS} />);

    expect(screen.getByText(/edit appearance/i)).toBeInTheDocument();
    expect(screen.getByText("Choose file")).toBeInTheDocument();
    expect(screen.getByText("Preview")).toBeInTheDocument();
    expect(screen.getByText("Fleet")).toBeInTheDocument();
    expect(screen.getByText("Self-service")).toBeInTheDocument();
    const save = screen.getByRole("button", { name: "Save" });
    expect(save).toBeInTheDocument();
  });

  it("shows the correct software name and preview info in Fleet card", () => {
    const render = createCustomRenderer({ withBackendMock: true });
    render(<EditIconModal {...MOCK_PROPS} />);
    expect(screen.getAllByText(software.name).length).toBeGreaterThan(0);
    expect(screen.getByText("Version")).toBeInTheDocument();
    expect(screen.getByText("Vulnerabilities")).toBeInTheDocument();
    expect(screen.getByText("88.0.1")).toBeInTheDocument();
    expect(screen.getByText("20 vulnerabilities")).toBeInTheDocument();
  });

  it("calls onExit handler when modal close is triggered", async () => {
    const render = createCustomRenderer({ withBackendMock: true });
    const { user } = render(<EditIconModal {...MOCK_PROPS} />);

    await user.keyboard("{Escape}");

    await waitFor(() => expect(MOCK_PROPS.onExit).toHaveBeenCalledTimes(1));
  });

  // Note: Rely on QA Wolf for E2e testing of file upload, preview, save, and remove icon

  describe("Display name tests", () => {
    it("shows the Display name input with correct default value", () => {
      const render = createCustomRenderer({ withBackendMock: true });
      render(<EditIconModal {...MOCK_PROPS} />);
      // Should default to blank if previewInfo.titleName === previewInfo.name
      const displayNameInput = screen.getByLabelText("Display name");
      expect(displayNameInput).toBeInTheDocument();
      expect(displayNameInput).toHaveValue("");
    });

    it("pre-fills Display name if previewInfo.name has been modified", () => {
      const MODIFIED_PROPS = {
        ...MOCK_PROPS,
        previewInfo: {
          ...MOCK_PROPS.previewInfo,
          name: "New Custom Name",
          titleName: "Original Title Name",
        },
      };
      const render = createCustomRenderer({ withBackendMock: true });
      render(<EditIconModal {...MODIFIED_PROPS} />);
      const displayNameInput = screen.getByLabelText("Display name");
      expect(displayNameInput).toBeInTheDocument();
      expect(displayNameInput).toHaveValue("New Custom Name");
      const helpText = screen.getByText(
        /Optional. If left blank, Fleet will use/
      );

      expect(helpText).toHaveTextContent(MODIFIED_PROPS.previewInfo.titleName);
    });

    it("only edits the display name when icon is not changed", async () => {
      const editSoftwarePackageSpy = jest
        .spyOn(softwareAPI, "editSoftwarePackage")
        .mockResolvedValue({});
      const deleteSoftwareIconSpy = jest.spyOn(
        softwareAPI,
        "deleteSoftwareIcon"
      );
      const editSoftwareIconSpy = jest.spyOn(softwareAPI, "editSoftwareIcon");

      const render = createCustomRenderer({ withBackendMock: true });
      const { user } = render(<EditIconModal {...MOCK_PROPS} />);

      const displayNameInput = screen.getByLabelText("Display name");
      await user.type(displayNameInput, "New Name     ");

      const saveButton = screen.getByRole("button", { name: "Save" });
      await user.click(saveButton);

      expect(editSoftwarePackageSpy).toHaveBeenCalledWith({
        data: { displayName: "New Name" }, // whitespace was trimmed
        // Multi-package titles require installer_id on every PATCH; the modal
        // targets `software_package` (mirror of `packages[0]`).
        installerId: softwarePackage.installer_id,
        softwareId: 123,
        teamId: 456,
      });

      expect(deleteSoftwareIconSpy).not.toHaveBeenCalled();
      expect(editSoftwareIconSpy).not.toHaveBeenCalled();
    });

    it("only edits the display name when removing a custom name", async () => {
      const editSoftwarePackageSpy = jest
        .spyOn(softwareAPI, "editSoftwarePackage")
        .mockResolvedValue({});

      const MODIFIED_PROPS = {
        ...MOCK_PROPS,
        previewInfo: {
          ...MOCK_PROPS.previewInfo,
          name: "Custom Display Name",
          titleName: "Original Software Name",
        },
      };

      const render = createCustomRenderer({ withBackendMock: true });
      const { user } = render(<EditIconModal {...MODIFIED_PROPS} />);

      const displayNameInput = screen.getByLabelText("Display name");
      await user.clear(displayNameInput);

      const saveButton = screen.getByRole("button", { name: "Save" });
      await user.click(saveButton);

      expect(editSoftwarePackageSpy).toHaveBeenCalledWith({
        data: { displayName: "" },
        installerId: softwarePackage.installer_id,
        softwareId: 123,
        teamId: 456,
      });
    });

    it("forwards the software package's installer_id on display-name save (#49239)", async () => {
      // Regression guard for the multi-package Edit-appearance flow: the
      // backend rejects display-name PATCHes without installer_id on titles
      // with multiple packages, so the modal must always send the id of the
      // package it's targeting (software_package == packages[0]).
      const editSoftwarePackageSpy = jest
        .spyOn(softwareAPI, "editSoftwarePackage")
        .mockResolvedValue({});

      const MULTI_PACKAGE_PROPS = {
        ...MOCK_PROPS,
        software: createMockSoftwarePackage({ installer_id: 42 }),
      };

      const render = createCustomRenderer({ withBackendMock: true });
      const { user } = render(<EditIconModal {...MULTI_PACKAGE_PROPS} />);

      const displayNameInput = screen.getByLabelText("Display name");
      await user.type(displayNameInput, "Custom label");

      const saveButton = screen.getByRole("button", { name: "Save" });
      await user.click(saveButton);

      expect(editSoftwarePackageSpy).toHaveBeenCalledWith(
        expect.objectContaining({ installerId: 42 })
      );
    });

    it("handles name update error properly", async () => {
      const editSoftwarePackageSpy = jest
        .spyOn(softwareAPI, "editSoftwarePackage")
        .mockRejectedValue(new Error("Name update failed"));

      const CUSTOM_ICON_PROPS = {
        ...MOCK_PROPS,
        previewInfo: {
          ...MOCK_PROPS.previewInfo,
          currentIconUrl: null,
        },
      };

      const render = createCustomRenderer({ withBackendMock: true });
      const { user } = render(<EditIconModal {...CUSTOM_ICON_PROPS} />);

      const displayNameInput = screen.getByLabelText("Display name");
      await user.type(displayNameInput, "New Name");

      const saveButton = screen.getByRole("button", { name: "Save" });
      await user.click(saveButton);

      expect(editSoftwarePackageSpy).toHaveBeenCalled();
      await expect(editSoftwarePackageSpy).rejects.toThrow(
        "Name update failed"
      );
    });
  });
});
