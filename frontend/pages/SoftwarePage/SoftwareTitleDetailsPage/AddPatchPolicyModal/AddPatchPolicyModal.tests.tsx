import React from "react";
import { screen } from "@testing-library/react";

import { noop } from "lodash";

import { createCustomRenderer } from "test/test-utils";
import AddPatchPolicyModal from "./AddPatchPolicyModal";

const renderModal = (props: { gitOpsModeEnabled?: boolean } = {}) => {
  const customRender = createCustomRenderer({
    context: {
      app: {
        config: {
          gitops: {
            gitops_mode_enabled: props.gitOpsModeEnabled ?? false,
          },
        },
      },
    },
  });

  return customRender(
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

  it("renders add button as disabled when gitOpsModeEnabled is true", async () => {
    const { user } = renderModal({ gitOpsModeEnabled: true });

    const addButton = screen.getByRole("button", { name: "Add" });
    expect(addButton).toBeDisabled();
    await user.hover(addButton);
  });
});
