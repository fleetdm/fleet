import React from "react";
import { screen } from "@testing-library/react";
import { renderWithSetup } from "test/test-utils";

import ConfirmDataCollectionDisableModal from "./ConfirmDataCollectionDisableModal";

describe("ConfirmDataCollectionDisableModal", () => {
  const renderModal = (
    overrides: Partial<
      React.ComponentProps<typeof ConfirmDataCollectionDisableModal>
    > = {}
  ) => {
    const onConfirm = jest.fn();
    const onCancel = jest.fn();
    const utils = renderWithSetup(
      <ConfirmDataCollectionDisableModal
        scope="global"
        datasets={["uptime"]}
        isUpdating={false}
        onConfirm={onConfirm}
        onCancel={onCancel}
        {...overrides}
      />
    );
    return { ...utils, onConfirm, onCancel };
  };

  it("renders the dataset labels prominently", () => {
    renderModal({ datasets: ["uptime", "vulnerabilities"] });
    expect(screen.getByText("Hosts active")).toBeInTheDocument();
    expect(screen.getByText("Vulnerabilities")).toBeInTheDocument();
  });

  it("uses global-scoped copy when scope is 'global'", () => {
    renderModal({ scope: "global" });
    expect(
      screen.getByText(/across this Fleet deployment/i)
    ).toBeInTheDocument();
  });

  it("uses fleet-scoped copy referencing the fleet name when scope is 'fleet'", () => {
    renderModal({ scope: "fleet", fleetName: "Engineering" });
    expect(screen.getByText(/Engineering/)).toBeInTheDocument();
  });

  it("calls onConfirm when Save and disable is clicked", async () => {
    const { user, onConfirm } = renderModal();
    await user.click(screen.getByRole("button", { name: /save and disable/i }));
    expect(onConfirm).toHaveBeenCalledTimes(1);
  });

  it("calls onCancel when Cancel is clicked", async () => {
    const { user, onCancel } = renderModal();
    await user.click(screen.getByRole("button", { name: /cancel/i }));
    expect(onCancel).toHaveBeenCalledTimes(1);
  });
});
