import React from "react";
import { render, screen } from "@testing-library/react";

import { noop } from "lodash";

import DeleteSoftwareModal from "./DeleteSoftwareModal";

const renderModal = (
  props: Partial<React.ComponentProps<typeof DeleteSoftwareModal>> = {}
) => {
  return render(
    <DeleteSoftwareModal
      softwareId={1}
      teamId={1}
      onExit={noop}
      onSuccess={noop}
      {...props}
    />
  );
};

describe("DeleteSoftwareModal", () => {
  beforeEach(() => {
    jest.resetAllMocks();
  });

  it("renders GitOps banner when gitOpsModeEnabled is true", () => {
    renderModal({ gitOpsModeEnabled: true });

    expect(
      screen.getByText(
        "You are currently in GitOps mode. If the package is defined in GitOps, it will reappear when GitOps runs."
      )
    ).toBeVisible();
  });

  it("renders default platform message when not VPP app or Android app", () => {
    renderModal();

    expect(screen.getByText(/won't be uninstalled/i)).toBeVisible();
    expect(
      screen.getByText(/Pending installs and uninstalls will be canceled\./i)
    ).toBeVisible();
  });

  it("renders App Store message when isAppStoreApp is true", () => {
    renderModal({ isAppStoreApp: true });

    expect(screen.getByText(/won't be uninstalled/i)).toBeVisible();
    expect(
      screen.getByText(
        /Pending or already started installs and uninstalls won't be canceled/i
      )
    ).toBeVisible();
  });

  it("renders Android message when isAndroidApp is true", () => {
    renderModal({ isAndroidApp: true });

    expect(screen.getByText(/will be uninstalled/i)).toBeVisible();
  });

  describe("multi-package title", () => {
    it("renders the 'Delete software' title and the custom-metadata warning by default (single-package legacy path)", () => {
      renderModal();

      expect(screen.getByText("Delete software")).toBeInTheDocument();
      expect(screen.queryByText("Delete package")).not.toBeInTheDocument();
      expect(
        screen.getByText("Custom icon and display name will be deleted.")
      ).toBeVisible();
    });

    it("renders the 'Delete package' title and suppresses the custom-metadata warning when canActivateMultiplePackages is true", () => {
      // On a multi-package title, only one installer is being deleted — the
      // title-level custom icon and display name stay put, so the warning
      // would be misleading.
      renderModal({ canActivateMultiplePackages: true });

      expect(screen.getByText("Delete package")).toBeInTheDocument();
      expect(screen.queryByText("Delete software")).not.toBeInTheDocument();
      expect(
        screen.queryByText("Custom icon and display name will be deleted.")
      ).not.toBeInTheDocument();
    });
  });
});
