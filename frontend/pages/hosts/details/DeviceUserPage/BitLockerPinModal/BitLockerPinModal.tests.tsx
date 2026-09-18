import { act, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import React from "react";

import { notify } from "components/ToastNotification";
import {
  DiskEncryptionActionRequired,
  IBitLockerPINRequest,
  IDeviceDiskEncryptionSetting,
} from "interfaces/host";
import { DiskEncryptionStatus } from "interfaces/mdm";
import diskEncryptionAPI from "services/entities/disk_encryption";

import BitLockerPinModal, { POLL_TIMEOUT_MS } from "./BitLockerPinModal";

jest.mock("services/entities/disk_encryption", () => ({
  __esModule: true,
  default: { submitBitLockerPIN: jest.fn() },
}));

jest.mock("components/ToastNotification", () => ({
  notify: { success: jest.fn(), error: jest.fn() },
}));

const submitBitLockerPIN = diskEncryptionAPI.submitBitLockerPIN as jest.Mock;

/** What a host still asking for a PIN reports, with the parts a test varies. */
const diskEncryption = (
  pinRequest?: IBitLockerPINRequest,
  status: DiskEncryptionStatus = "action_required",
  actionRequired: DiskEncryptionActionRequired = "create_pin"
): IDeviceDiskEncryptionSetting => ({
  status,
  detail: "",
  action_required: actionRequired,
  pin_request: pinRequest,
});

/** Renders the modal and hands back a way to feed it the page's next fetch. */
const renderModal = (onExit = jest.fn()) => {
  // delay: null keeps user-event off the fake clock, which otherwise makes typing and clicking flaky under load.
  const user = userEvent.setup({
    advanceTimers: jest.advanceTimersByTime,
    delay: null,
  });
  const onSubmitted = jest.fn();
  const props = {
    deviceAuthToken: "token",
    diskEncryption: diskEncryption(),
    // Before any submit, so nothing already on screen counts as this submit's answer.
    dataUpdatedAt: 0,
    onSubmitted,
    onExit,
  };
  const { rerender } = render(<BitLockerPinModal {...props} />);

  /** Stands in for the page's query resolving: new data, fetched now. */
  const pageFetched = (next: IDeviceDiskEncryptionSetting) =>
    act(() => {
      // Fake timers freeze Date.now(), so move it on: data fetched in the same millisecond as the submit cannot be
      // this submit's answer, and the modal is right to ignore it.
      jest.advanceTimersByTime(1000);
      rerender(
        <BitLockerPinModal
          {...props}
          diskEncryption={next}
          dataUpdatedAt={Date.now()}
        />
      );
    });

  return { user, onExit, onSubmitted, pageFetched };
};

const submitPIN = async (
  user: ReturnType<typeof userEvent.setup>,
  pin: string,
  confirmPin = pin
) => {
  await user.type(screen.getByLabelText("BitLocker PIN"), pin);
  await user.type(screen.getByLabelText("Confirm PIN"), confirmPin);
  await user.click(screen.getByRole("button", { name: "Save" }));
  // Let the submit request settle so the modal starts waiting. advanceTimersByTimeAsync drains the microtask queue,
  // which a single awaited Promise does not: the click, the request and the state update are several ticks apart.
  await act(async () => {
    await jest.advanceTimersByTimeAsync(0);
  });
};

describe("BitLockerPinModal", () => {
  beforeEach(() => {
    jest.clearAllMocks();
    jest.useFakeTimers();
    submitBitLockerPIN.mockResolvedValue(undefined);
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  it("holds back a PIN the server would reject", async () => {
    const { user } = renderModal();

    await submitPIN(user, "12345");

    expect(screen.getByText("Use 6 to 20 characters")).toBeVisible();
    expect(submitBitLockerPIN).not.toHaveBeenCalled();
  });

  it("holds back a PIN the end user did not confirm", async () => {
    const { user } = renderModal();

    await submitPIN(user, "123456", "123457");

    expect(screen.getByText("PINs must match")).toBeVisible();
    expect(submitBitLockerPIN).not.toHaveBeenCalled();
  });

  it("asks the page to refetch so the wait has something to read", async () => {
    const { user, onSubmitted } = renderModal();

    // Spaces are part of a BitLocker PIN, so the modal must not trim them away.
    await submitPIN(user, " pin 1234 ");

    expect(submitBitLockerPIN).toHaveBeenCalledWith("token", " pin 1234 ");
    expect(onSubmitted).toHaveBeenCalled();
    expect(screen.getByText("Setting PIN...")).toBeVisible();
  });

  it("closes with a success toast once the agent reports the PIN is set", async () => {
    const { user, onExit, pageFetched } = renderModal();

    await submitPIN(user, "123456");
    pageFetched(diskEncryption({ status: "set", error: "" }));

    expect(notify.success).toHaveBeenCalledWith("Successfully created PIN.");
    expect(onExit).toHaveBeenCalled();
  });

  it("closes with a success toast when osquery sees the PIN before the agent reports it", async () => {
    const { user, onExit, pageFetched } = renderModal();

    await submitPIN(user, "123456");
    pageFetched(
      diskEncryption({ status: "delivered", error: "" }, "verified", undefined)
    );

    expect(notify.success).toHaveBeenCalledWith("Successfully created PIN.");
    expect(onExit).toHaveBeenCalled();
  });

  it("keeps waiting while the agent has not answered", async () => {
    const { user, onExit, pageFetched } = renderModal();

    await submitPIN(user, "123456");
    pageFetched(diskEncryption({ status: "delivered", error: "" }));

    expect(screen.getByText("Setting PIN...")).toBeVisible();
    expect(notify.success).not.toHaveBeenCalled();
    expect(notify.error).not.toHaveBeenCalled();
    expect(onExit).not.toHaveBeenCalled();
  });

  it("ignores an outcome fetched before the submit", async () => {
    // A failure from an earlier attempt is still in the page's data when this one is submitted.
    const { user, onExit } = renderModal();

    await submitPIN(user, "123456");

    expect(notify.error).not.toHaveBeenCalled();
    expect(onExit).not.toHaveBeenCalled();
    expect(screen.getByText("Setting PIN...")).toBeVisible();
  });

  it("stays open and reports the agent's reason when the PIN is refused", async () => {
    const { user, onExit, pageFetched } = renderModal();

    await submitPIN(user, "123456");
    pageFetched(
      diskEncryption({ status: "failed", error: "PIN already set." })
    );

    expect(notify.error).toHaveBeenCalledWith(
      "Couldn't set PIN. PIN already set. Try again or contact your IT admin."
    );
    expect(onExit).not.toHaveBeenCalled();
    // Save is offered again, so the button is out of its waiting state.
    expect(screen.getByRole("button", { name: "Save" })).toBeEnabled();
  });

  it("reports the agent's failure even once the host stops asking for a PIN", async () => {
    const { user, onExit, pageFetched } = renderModal();

    await submitPIN(user, "123456");
    // A host can stop asking for a PIN for reasons unrelated to this submission.
    pageFetched(
      diskEncryption(
        { status: "failed", error: "PIN already set" },
        "action_required",
        "restart"
      )
    );

    expect(notify.error).toHaveBeenCalledWith(
      "Couldn't set PIN. PIN already set. Try again or contact your IT admin."
    );
    expect(notify.success).not.toHaveBeenCalled();
    expect(onExit).not.toHaveBeenCalled();
  });

  it("closes without claiming failure when the agent has not answered in time", async () => {
    const { user, onExit } = renderModal();

    await submitPIN(user, "123456");
    act(() => {
      jest.advanceTimersByTime(POLL_TIMEOUT_MS);
    });

    // The server keeps the request collectable for far longer, so the PIN may still land.
    expect(notify.error).toHaveBeenCalledWith(
      "PIN submitted but Fleet is still working on it. You’ll see an update when this device responds."
    );
    expect(notify.success).not.toHaveBeenCalled();
    expect(onExit).toHaveBeenCalled();
  });

  it("reports the server's reason for a rejected submission without waiting", async () => {
    const reason =
      "Fleet's agent on this host is too old to set a BitLocker PIN. Update fleetd and try again.";
    submitBitLockerPIN.mockRejectedValue({
      response: { data: { errors: [{ name: "base", reason }] } },
    });
    const { user, onExit, onSubmitted } = renderModal();

    await submitPIN(user, "123456");

    expect(notify.error).toHaveBeenCalledWith(
      `Couldn't set PIN. ${reason} Try again or contact your IT admin.`,
      expect.anything()
    );
    // A rejected submission is never in the agent's hands, so the page is not asked to poll for it.
    expect(onSubmitted).not.toHaveBeenCalled();
    expect(onExit).not.toHaveBeenCalled();
  });
});
