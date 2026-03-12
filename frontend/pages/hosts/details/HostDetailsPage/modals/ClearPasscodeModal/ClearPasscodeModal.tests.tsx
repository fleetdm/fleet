import React from "react";
import { screen } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";

import ClearPasscodeModal from "./ClearPasscodeModal";

const MOCK_PROPS = {
  id: 7,
  hostName: "iphone-host-1",
  onSuccess: jest.fn(),
  onClose: jest.fn(),
};

describe("ClearPasscodeModal", () => {
  beforeEach(() => {
    jest.resetAllMocks();
  });

  it("renders description and confirmation text", () => {
    const render = createCustomRenderer({ withBackendMock: true });
    render(<ClearPasscodeModal {...MOCK_PROPS} />);

    expect(
      screen.getByText(
        /Clearing the passcode allows the user to set a new passcode on the device./i
      )
    ).toBeInTheDocument();
    expect(screen.getByText(/I wish to clear the passcode on/i)).toBeInTheDocument();
    expect(screen.getByText(/iphone-host-1/i)).toBeInTheDocument();
  });

  it("disables Clear passcode button until confirm checkbox is checked", async () => {
    const render = createCustomRenderer({ withBackendMock: true });
    const { user } = render(<ClearPasscodeModal {...MOCK_PROPS} />);

    const clearButton = screen.getByRole("button", { name: /Clear passcode/i });
    const cancelButton = screen.getByRole("button", { name: "Cancel" });

    expect(clearButton).toBeDisabled();
    expect(cancelButton).toBeEnabled();

    const checkbox = screen.getByRole("checkbox", {
      name: /iphone-host-1/i,
    });

    await user.click(checkbox);

    expect(clearButton).toBeEnabled();
  });

  it("calls onClose when Cancel is clicked", async () => {
    const render = createCustomRenderer({ withBackendMock: true });
    const { user } = render(<ClearPasscodeModal {...MOCK_PROPS} />);

    const cancelButton = screen.getByRole("button", { name: "Cancel" });
    await user.click(cancelButton);

    expect(MOCK_PROPS.onClose).toHaveBeenCalled();
  });
});
