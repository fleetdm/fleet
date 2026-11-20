import { screen, waitFor } from "@testing-library/react";
import React from "react";

import hostAPI from "services/entities/hosts";
import { createCustomRenderer } from "test/test-utils";

import ViewUnlockPinModal from "./ViewUnlockPinModal";

jest.mock("services/entities/hosts");

const LOCKED_AT = "2026-10-23T14:35:34Z";

describe("ViewUnlockPinModal", () => {
  const render = createCustomRenderer({ withBackendMock: true });

  beforeEach(() => {
    jest.resetAllMocks();
    (hostAPI.unlockHost as jest.Mock).mockResolvedValue({
      host_id: 7,
      unlock_pin: "123456",
      locked_at: LOCKED_AT,
    });
  });

  it("shows a spinner and a Close button while loading", async () => {
    (hostAPI.unlockHost as jest.Mock).mockReturnValue(new Promise(() => undefined));
    render(<ViewUnlockPinModal id={7} onClose={jest.fn()} />);

    // Spinner waits 250ms before rendering to avoid a flash.
    expect(await screen.findByTestId("spinner")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Close" })).toBeInTheDocument();
  });

  it("shows an error state when the request fails", async () => {
    (hostAPI.unlockHost as jest.Mock).mockRejectedValue(new Error("nope"));
    render(<ViewUnlockPinModal id={7} onClose={jest.fn()} />);

    await waitFor(() => {
      expect(screen.getByText(/something's gone wrong/i)).toBeInTheDocument();
    });
    expect(screen.queryByText("123456")).not.toBeInTheDocument();
  });

  it("renders the lock time, PIN, help text, and a single Close button", async () => {
    const onClose = jest.fn();
    const { user } = render(<ViewUnlockPinModal id={7} onClose={onClose} />);

    await waitFor(() => {
      expect(screen.getByText("123456")).toBeInTheDocument();
    });
    expect(screen.getByText(/Fleet locked this host/)).toBeInTheDocument();
    expect(screen.getByText("PIN:")).toBeInTheDocument();
    expect(
      screen.getByText(/If the PIN doesn't work, it was changed/)
    ).toBeInTheDocument();
    expect(screen.getByText(/Clear passcode/)).toBeInTheDocument();

    // The modal's own "x" is labelled "Exit"; there is exactly one Close.
    const closeButtons = screen.getAllByRole("button", { name: "Close" });
    expect(closeButtons).toHaveLength(1);
    await user.click(closeButtons[0]);
    expect(onClose).toHaveBeenCalled();
  });

  it("shows the full timestamp when hovering the relative time", async () => {
    const { user } = render(<ViewUnlockPinModal id={7} onClose={jest.fn()} />);

    await waitFor(() => {
      expect(screen.getByText("123456")).toBeInTheDocument();
    });
    const relativeTime = screen.getByText(/ago$/);
    await user.hover(relativeTime);

    await waitFor(() => {
      expect(screen.getByText(/10\/23\/2026|2026/)).toBeInTheDocument();
    });
  });

  it("omits the lock time when locked_at is missing", async () => {
    (hostAPI.unlockHost as jest.Mock).mockResolvedValue({
      host_id: 7,
      unlock_pin: "123456",
    });
    render(<ViewUnlockPinModal id={7} onClose={jest.fn()} />);

    await waitFor(() => {
      expect(screen.getByText("123456")).toBeInTheDocument();
    });
    expect(
      screen.queryByText(/Fleet locked this host/)
    ).not.toBeInTheDocument();
  });

  it("sends a new request each time the modal is opened", async () => {
    const first = render(<ViewUnlockPinModal id={7} onClose={jest.fn()} />);
    await waitFor(() => {
      expect(screen.getByText("123456")).toBeInTheDocument();
    });
    first.unmount();

    render(<ViewUnlockPinModal id={7} onClose={jest.fn()} />);
    await waitFor(() => {
      expect(hostAPI.unlockHost).toHaveBeenCalledTimes(2);
    });
  });
});
