import React from "react";
import { render, screen } from "@testing-library/react";

import { noop } from "lodash";

import AddPatchPolicyModal from "./AddPatchPolicyModal";

const renderModal = (
  props: Partial<React.ComponentProps<typeof AddPatchPolicyModal>> = {}
) => {
  return render(
    <AddPatchPolicyModal
      softwareId={1}
      teamId={1}
      onExit={noop}
      onSuccess={noop}
      {...props}
    />
  );
};

describe("AddPatchPolicyModal", () => {
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
});
