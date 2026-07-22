import React from "react";

import { createCustomRenderer } from "test/test-utils";
import { screen } from "@testing-library/react";
import ReleaseFromABModal, {
  IReleaseFromABModalProps,
} from "./ReleaseFromABModal";

describe("Release from AB modal", () => {
  const renderComponent = (props: IReleaseFromABModalProps) => {
    return createCustomRenderer({ context: {}, withBackendMock: true })(
      <ReleaseFromABModal {...props} />
    );
  };
  it("disables release until checkbox is checked", async () => {
    const { user } = renderComponent({
      host: { id: 1, display_name: "Test Host" },
      onExit: jest.fn(),
      onRelease: jest.fn(),
    });

    const releaseButton = screen.getByText("Release").closest("button");
    expect(releaseButton).toBeDisabled();

    const checkbox = screen.getByRole("checkbox", {
      name: /I understand this action can't be undone/i,
    });
    await user.click(checkbox);

    expect(releaseButton).toBeEnabled();
  });
});
