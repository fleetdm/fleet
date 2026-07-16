import React from "react";
import { screen } from "@testing-library/react";

import { createCustomRenderer } from "test/test-utils";

import AddPackageModal from "./AddPackageModal";

const BASE_PROPS = {
  softwareTitleId: 42,
  softwareTitleName: "GlobalProtect",
  teamId: 1,
  existingPackageName: "GlobalProtect-v6.3.2.pkg",
  onExit: jest.fn(),
  onSuccess: jest.fn(),
};

const renderModal = (
  overrides: Partial<React.ComponentProps<typeof AddPackageModal>> = {},
  gitOpsModeEnabled = false
) => {
  const render = createCustomRenderer({
    withBackendMock: true,
    context: {
      app: {
        isPremiumTier: true,
        isGlobalAdmin: true,
        config: {
          gitops: {
            gitops_mode_enabled: gitOpsModeEnabled,
            repository_url: gitOpsModeEnabled ? "https://example.com/repo" : "",
          },
        },
      },
    },
  });
  return render(<AddPackageModal {...BASE_PROPS} {...overrides} />);
};

describe("AddPackageModal", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe("standard mode", () => {
    it("renders with the 'Add package' title", () => {
      renderModal();
      // Modal renders its title as a <span>, not a heading; use getByText.
      expect(screen.getByText("Add package")).toBeInTheDocument();
    });

    it("renders the multi-package first-added banner in the target section", () => {
      renderModal();
      expect(
        screen.getByText(
          /If multiple packages of the same software target the same host, Fleet will install the one that was added first\./i
        )
      ).toBeInTheDocument();
    });

    it("hides the GitOps banner copy", () => {
      renderModal();
      expect(
        screen.queryByText(/Add custom packages in GitOps mode/i)
      ).not.toBeInTheDocument();
      expect(screen.queryByText("YAML docs")).not.toBeInTheDocument();
    });

    it("derives the platform label from the existing package's filename", () => {
      renderModal({ existingPackageName: "GlobalProtect-v6.3.2.pkg" });
      // `getFileTypeRestriction` returns label "macOS (.pkg)" — the modal
      // forwards it to PackageForm's FileUploader message slot.
      expect(screen.getByText("macOS (.pkg)")).toBeInTheDocument();
    });

    it("falls back to the all-platforms file-type message when the existing name has no recognized extension", () => {
      renderModal({ existingPackageName: "no-extension" });
      // PackageForm's default message lists every supported platform.
      expect(screen.getByText(/macOS \(.pkg,/)).toBeInTheDocument();
    });

    it("renders the form's Save button as 'Save' (not 'Add software')", async () => {
      renderModal();
      // The button text comes from PackageForm — `multiPackageContext` flips
      // it from "Add software" to "Save". The form mounts after labels load
      // (an empty array via the optional-chained fallback), so await it.
      const saveButton = await screen.findByRole("button", { name: "Save" });
      expect(saveButton).toBeInTheDocument();
      expect(
        screen.queryByRole("button", { name: "Add software" })
      ).not.toBeInTheDocument();
    });

    it("preselects the Custom target radio (multi-package default per Figma)", () => {
      renderModal();
      const customRadio = screen.getByLabelText("Custom");
      expect(customRadio).toBeChecked();
    });
  });

  describe("GitOps mode", () => {
    it("renders the GitOps banner copy", () => {
      renderModal({}, true);
      expect(
        screen.getByText(/Add custom packages in GitOps mode/i)
      ).toBeInTheDocument();
      expect(
        screen.getByText(
          /copy its SHA-256 hash into your YAML so the next GitOps workflow doesn.t delete it/i
        )
      ).toBeInTheDocument();
    });

    it("renders the YAML docs CustomLink", () => {
      renderModal({}, true);
      const link = screen.getByRole("link", { name: /YAML docs/i });
      expect(link).toHaveAttribute(
        "href",
        expect.stringMatching(/learn-more-about\/software-yaml$/)
      );
    });

    it("hides the standard multi-package banner copy in GitOps mode", () => {
      renderModal({}, true);
      expect(
        screen.queryByText(/will install the one that was added first/)
      ).not.toBeInTheDocument();
    });
  });

  describe("file-type restriction (per-row)", () => {
    it("constrains a Linux .deb title to .deb uploads", () => {
      renderModal({ existingPackageName: "cinc_18.2.11-1_amd64.deb" });
      expect(screen.getByText("Linux (.deb)")).toBeInTheDocument();
    });

    it("constrains a Windows .msi title to .msi uploads", () => {
      renderModal({ existingPackageName: "ZoomInstaller.msi" });
      expect(screen.getByText("Windows (.msi)")).toBeInTheDocument();
    });

    it("constrains a .sh script-only title to .sh uploads", () => {
      renderModal({ existingPackageName: "setup.sh" });
      expect(screen.getByText("macOS & Linux (.sh)")).toBeInTheDocument();
    });
  });
});
