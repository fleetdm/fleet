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

/** A host the agent has taken a PIN for and not reported on yet. */
const inTheAgentsHands = diskEncryption({ status: "delivered", error: "" });

/** Renders the modal and hands back a way to feed it the page's next fetch. */
const renderModal = (beforeSubmit = diskEncryption()) => {
  // delay: null keeps user-event off the fake clock, which otherwise makes typing and clicking flaky under load.
  const user = userEvent.setup({
    advanceTimers: jest.advanceTimersByTime,
    delay: null,
  });
  const onWaitingChange = jest.fn();
  const onExit = jest.fn();
  // onExit is inline here as it is on the page, so every render hands the modal a different function. A modal that
  // took that as a signal would restart its deadline on every poll.
  const modal = (data: IDeviceDiskEncryptionSetting, dataUpdatedAt: number) => (
    <BitLockerPinModal
      deviceAuthToken="token"
      diskEncryption={data}
      dataUpdatedAt={dataUpdatedAt}
      onWaitingChange={onWaitingChange}
      onExit={() => onExit()}
    />
  );
  // Fetched before any submit, so nothing already on screen counts as this submit's answer.
  const { rerender, unmount } = render(modal(beforeSubmit, 0));

  /** Stands in for the page's query resolving: new data, fetched now. */
  const pageFetched = (next: IDeviceDiskEncryptionSetting) =>
    act(() => {
      // Fake timers freeze Date.now(), so move it on: data fetched in the same millisecond as the submit cannot be
      // this submit's answer, and the modal is right to ignore it.
      jest.advanceTimersByTime(1000);
      rerender(modal(next, Date.now()));
    });

  return { user, onExit, onWaitingChange, pageFetched, unmount };
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

/** Every signal the modal accepts as the PIN having been set. */
const successCases: [string, IDeviceDiskEncryptionSetting][] = [
  [
    "the agent reports the PIN is set",
    diskEncryption({ status: "set", error: "" }),
  ],
  // osquery can see the PIN before the agent's own report reaches Fleet.
  [
    "the disk reads as verified",
    diskEncryption({ status: "delivered", error: "" }, "verified", undefined),
  ],
  [
    "the disk reads as verifying",
    diskEncryption({ status: "delivered", error: "" }, "verifying", undefined),
  ],
];

const stillWaitingCases: [string, IDeviceDiskEncryptionSetting][] = [
  ["the agent has not answered", inTheAgentsHands],
  // The request is read from a replica, so the fetch right after a submit can still show nothing in flight.
  ["a fetch answers from before the submit landed", diskEncryption()],
];

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

  it("tells the page it is waiting, so the page fetches the outcome", async () => {
    const { user, onWaitingChange } = renderModal();

    // Spaces are part of a BitLocker PIN, so the modal must not trim them away.
    await submitPIN(user, " pin 1234 ");

    expect(submitBitLockerPIN).toHaveBeenCalledWith("token", " pin 1234 ");
    expect(onWaitingChange).toHaveBeenLastCalledWith(true);
    expect(screen.getByText("Setting PIN...")).toBeVisible();
  });

  it.each(successCases)(
    "closes with a success toast once %s",
    async (_, fetched) => {
      const { user, onExit, onWaitingChange, pageFetched } = renderModal();

      await submitPIN(user, "123456");
      pageFetched(fetched);

      expect(notify.success).toHaveBeenCalledWith("Successfully created PIN.");
      expect(onExit).toHaveBeenCalled();
      expect(onWaitingChange).toHaveBeenLastCalledWith(false);
    }
  );

  it.each(stillWaitingCases)("keeps waiting while %s", async (_, fetched) => {
    const { user, onExit, onWaitingChange, pageFetched } = renderModal();

    await submitPIN(user, "123456");
    pageFetched(fetched);

    expect(screen.getByText("Setting PIN...")).toBeVisible();
    expect(notify.success).not.toHaveBeenCalled();
    expect(notify.error).not.toHaveBeenCalled();
    expect(onExit).not.toHaveBeenCalled();
    expect(onWaitingChange).toHaveBeenLastCalledWith(true);
  });

  it("ignores an outcome fetched before the submit", async () => {
    // A failure from an earlier attempt is still in the page's data when this one is submitted.
    const { user, onExit } = renderModal(
      diskEncryption({ status: "failed", error: "PIN already set" })
    );

    await submitPIN(user, "123456");

    expect(notify.error).not.toHaveBeenCalled();
    expect(onExit).not.toHaveBeenCalled();
    expect(screen.getByText("Setting PIN...")).toBeVisible();
  });

  it("stays open and reports the agent's reason when the PIN is refused", async () => {
    const { user, onExit, onWaitingChange, pageFetched } = renderModal();

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
    // The wait is over even though the modal stays open, so the page stops polling.
    expect(onWaitingChange).toHaveBeenLastCalledWith(false);
  });

  it("reports the agent's failure ahead of any success signal", async () => {
    const { user, onExit, pageFetched } = renderModal();

    await submitPIN(user, "123456");
    // Both signals at once. A host can be encrypting for reasons unrelated to this submission, so success needs the
    // agent's own report and cannot be read off the disk encryption status alone.
    pageFetched(
      diskEncryption(
        { status: "failed", error: "PIN already set" },
        "verifying",
        undefined
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

  it("holds that deadline while the page polls", async () => {
    const { user, onExit, pageFetched } = renderModal();

    await submitPIN(user, "123456");
    // Every poll re-renders the modal. A deadline that restarted with them would never be reached.
    for (let elapsed = 0; elapsed <= POLL_TIMEOUT_MS; elapsed += 1000) {
      pageFetched(inTheAgentsHands);
    }

    expect(onExit).toHaveBeenCalled();
  });

  it("drops the wait when the modal closes mid-flight", async () => {
    const { user, unmount } = renderModal();

    await submitPIN(user, "123456");
    unmount();
    act(() => {
      jest.advanceTimersByTime(POLL_TIMEOUT_MS);
    });

    // Nothing is left running to report on a modal the end user has already closed.
    expect(notify.error).not.toHaveBeenCalled();
    expect(notify.success).not.toHaveBeenCalled();
  });

  it("reports the server's reason for a rejected submission without waiting", async () => {
    const reason =
      "Fleet's agent on this host is too old to set a BitLocker PIN. Update fleetd and try again.";
    submitBitLockerPIN.mockRejectedValue({
      response: { data: { errors: [{ name: "base", reason }] } },
    });
    const { user, onExit, onWaitingChange } = renderModal();

    await submitPIN(user, "123456");

    expect(notify.error).toHaveBeenCalledWith(
      `Couldn't set PIN. ${reason} Try again or contact your IT admin.`,
      expect.anything()
    );
    // A rejected submission is never in the agent's hands, so the page is not asked to poll for it.
    expect(onWaitingChange).not.toHaveBeenCalledWith(true);
    expect(onExit).not.toHaveBeenCalled();
  });
});
