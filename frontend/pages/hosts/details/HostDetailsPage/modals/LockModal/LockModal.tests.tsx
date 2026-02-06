import React from "react";
import { screen } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";

import LockModal from "./LockModal";

const MOCK_PROPS = {
  id: 7,
  platform: "darwin",
  hostName: "macos-host-1",
  onSuccess: jest.fn(),
  onClose: jest.fn(),
};

describe("LockModal", () => {
  beforeEach(() => {
    jest.resetAllMocks();
  });

  it("renders macOS description and confirm text for non‑iOS", () => {
    const render = createCustomRenderer({ withBackendMock: true });
    render(<LockModal {...MOCK_PROPS} />);

    expect(
      screen.getByText(
        /Lock a host when it needs to be returned to your organization./i
      )
    ).toBeInTheDocument();
    expect(
      screen.getByText(/Fleet will generate a six-digit unlock PIN./i)
    ).toBeInTheDocument();
    expect(screen.getByText(/I wish to lock/i)).toBeInTheDocument();
    expect(screen.getByText(/macos-host-1/i)).toBeInTheDocument();
  });

  it("renders iOS Lost Mode description and preview card when platform is iOS/iPadOS", () => {
    const render = createCustomRenderer({ withBackendMock: true });
    render(<LockModal {...MOCK_PROPS} platform="ios" hostName="iphone-1" />);

    expect(
      screen.getByText(/This enables what Apple calls/i)
    ).toBeInTheDocument();
    expect(screen.getByText(/Lost Mode/i)).toBeInTheDocument();
    expect(screen.getByText(/End user experience/i)).toBeInTheDocument();
    expect(
      screen.getByAltText(/iPhone with a lock screen message/i)
    ).toBeInTheDocument();
    expect(screen.getByText(/I wish to lock/i)).toBeInTheDocument();
    expect(screen.getByText(/iphone-1/i)).toBeInTheDocument();
  });

  it("disables Lock button until confirm checkbox is checked", async () => {
    const render = createCustomRenderer({ withBackendMock: true });
    const { user } = render(<LockModal {...MOCK_PROPS} />);

    const lockButton = screen.getByRole("button", { name: "Lock" });
    const cancelButton = screen.getByRole("button", { name: "Cancel" });

    expect(lockButton).toBeDisabled();
    expect(cancelButton).toBeEnabled();

    const checkbox = screen.getByRole("checkbox", {
      name: /macos-host-1/i,
    });

    await user.click(checkbox);

    expect(lockButton).toBeEnabled();
  });

  it("calls onClose when Cancel is clicked", async () => {
    const render = createCustomRenderer({ withBackendMock: true });
    const { user } = render(<LockModal {...MOCK_PROPS} />);

    const cancelButton = screen.getByRole("button", { name: "Cancel" });
    await user.click(cancelButton);

    expect(MOCK_PROPS.onClose).toHaveBeenCalled();
  });
});
