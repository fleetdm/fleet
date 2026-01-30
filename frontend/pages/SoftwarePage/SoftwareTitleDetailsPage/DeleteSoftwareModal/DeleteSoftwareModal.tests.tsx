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

    expect(
      screen.getByText(
        /Currently, software won't be deleted from self-service \(managed Google Play\) and won't be uninstalled from the hosts\./i
      )
    ).toBeVisible();
  });
});
