import React from "react";
import { screen } from "@testing-library/react";
import { renderWithSetup } from "test/test-utils";

import RotationFailedDetailsModal from "./RotationFailedDetailsModal";

const REASON =
  "Resetting password for _fleetadmin failed: The password does not meet the password policy requirements.";

describe("RotationFailedDetailsModal", () => {
  it("names the host in the failure sentence", () => {
    renderWithSetup(
      <RotationFailedDetailsModal
        detail={REASON}
        hostDisplayName="DESKTOP-ABC123"
        onCancel={jest.fn()}
      />
    );

    expect(screen.getByText("Rotation details")).toBeVisible();
    expect(
      screen.getByText(
        /failed to rotate the managed local account password on/i
      )
    ).toBeVisible();
    expect(screen.getByText("DESKTOP-ABC123")).toBeVisible();
  });

  it("falls back to 'the host' when there is no display name", () => {
    renderWithSetup(
      <RotationFailedDetailsModal
        detail={REASON}
        hostDisplayName=""
        onCancel={jest.fn()}
      />
    );

    expect(screen.getByText(/password on the host\./i)).toBeVisible();
  });

  // Same shape as the other failed-install modals: the reason is behind a toggle, not shown up front.
  it("hides the reason until Details is opened", async () => {
    const { user } = renderWithSetup(
      <RotationFailedDetailsModal
        detail={REASON}
        hostDisplayName="DESKTOP-ABC123"
        onCancel={jest.fn()}
      />
    );

    expect(screen.queryByText(REASON)).not.toBeInTheDocument();
    expect(screen.queryByText("Error details:")).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /details/i }));

    expect(screen.getByText("Error details:")).toBeVisible();
    expect(screen.getByText(REASON)).toBeVisible();
  });

  it("offers no Details toggle when the host gave no reason", () => {
    renderWithSetup(
      <RotationFailedDetailsModal
        detail=""
        hostDisplayName="DESKTOP-ABC123"
        onCancel={jest.fn()}
      />
    );

    expect(
      screen.queryByRole("button", { name: /details/i })
    ).not.toBeInTheDocument();
  });

  it("calls onCancel from Close", async () => {
    const onCancel = jest.fn();
    const { user } = renderWithSetup(
      <RotationFailedDetailsModal
        detail={REASON}
        hostDisplayName="DESKTOP-ABC123"
        onCancel={onCancel}
      />
    );

    await user.click(screen.getByRole("button", { name: "Close" }));

    expect(onCancel).toHaveBeenCalledTimes(1);
  });
});
