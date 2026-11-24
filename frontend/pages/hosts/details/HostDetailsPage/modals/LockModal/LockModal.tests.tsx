import { screen } from "@testing-library/react";
import React from "react";

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
        /During offboarding, lock a host when it needs to be returned to your organization./i
      )
    ).toBeInTheDocument();
    expect(
      screen.getByText(/Fleet will generate a six-digit unlock PIN./i)
    ).toBeInTheDocument();
    expect(screen.getByText(/I wish to lock/i)).toBeInTheDocument();
    expect(screen.getByText(/macos-host-1/i)).toBeInTheDocument();
  });

  it("renders the iOS description and preview card without Lost Mode copy", () => {
    const render = createCustomRenderer({ withBackendMock: true });
    render(<LockModal {...MOCK_PROPS} platform="ios" hostName="iphone-1" />);

    expect(
      screen.getByText(
        /During offboarding, lock a host when it needs to be returned to your organization./i
      )
    ).toBeInTheDocument();
    expect(screen.queryByText(/Lost Mode/i)).not.toBeInTheDocument();
    expect(
      screen.queryByText(/It can only be unlocked through Fleet./i)
    ).not.toBeInTheDocument();
    expect(screen.getByText(/End user experience/i)).toBeInTheDocument();
    expect(
      screen.getByAltText(/iPhone with a lock screen message/i)
    ).toBeInTheDocument();
    expect(screen.getByText(/I wish to lock/i)).toBeInTheDocument();
    expect(screen.getByText(/iphone-1/i)).toBeInTheDocument();
  });

  it.each(["windows", "ubuntu"])(
    "renders only the offboarding line for %s",
    (platform) => {
      const render = createCustomRenderer({ withBackendMock: true });
      render(<LockModal {...MOCK_PROPS} platform={platform} />);

      expect(
        screen.getByText(
          /During offboarding, lock a host when it needs to be returned to your organization./i
        )
      ).toBeInTheDocument();
      expect(screen.queryByText(/unlock PIN/i)).not.toBeInTheDocument();
    }
  );

  it("renders the Android BYOD copy", () => {
    const render = createCustomRenderer({ withBackendMock: true });
    render(
      <LockModal
        {...MOCK_PROPS}
        platform="android"
        hostMdmEnrollmentStatus="On (personal)"
      />
    );

    expect(
      screen.getByText("Lock the end user out of the work profile.")
    ).toBeInTheDocument();
    expect(
      screen.getByText(/Fleet will generate an unlock PIN. To view it, select/)
    ).toBeInTheDocument();
    expect(screen.getByText("Actions > View unlock PIN")).toBeInTheDocument();
    expect(screen.queryByText(/During offboarding/)).not.toBeInTheDocument();
    expect(screen.queryByText(/six-digit/)).not.toBeInTheDocument();
  });

  it("renders the Android company-owned copy", () => {
    const render = createCustomRenderer({ withBackendMock: true });
    render(
      <LockModal
        {...MOCK_PROPS}
        platform="android"
        hostMdmEnrollmentStatus="On (automatic)"
      />
    );

    expect(
      screen.getByText(
        /During offboarding, lock a host when it needs to be returned to your organization./i
      )
    ).toBeInTheDocument();
    expect(
      screen.getByText(/Fleet will generate an unlock PIN. To view it, select/)
    ).toBeInTheDocument();
    expect(screen.getByText("Actions > View unlock PIN")).toBeInTheDocument();
    expect(
      screen.queryByText("Lock the end user out of the work profile.")
    ).not.toBeInTheDocument();
  });

  it("renders WiFi warning for iOS/iPadOS platforms", () => {
    const render = createCustomRenderer({ withBackendMock: true });
    render(<LockModal {...MOCK_PROPS} platform="ios" hostName="iphone-1" />);

    expect(
      screen.getByText(
        /If the host is turned off or restarted while locked, it will disconnect from Wi-Fi, and you won't be able to unlock it remotely/i
      )
    ).toBeInTheDocument();
  });

  it("does not render WiFi warning for macOS platform", () => {
    const render = createCustomRenderer({ withBackendMock: true });
    render(<LockModal {...MOCK_PROPS} platform="darwin" />);

    expect(
      screen.queryByText(
        /If the device is turned off or restarted while locked/i
      )
    ).not.toBeInTheDocument();
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
