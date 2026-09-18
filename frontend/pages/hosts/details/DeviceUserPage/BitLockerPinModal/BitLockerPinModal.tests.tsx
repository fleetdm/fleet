import { act, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import React from "react";

import createMockHost from "__mocks__/hostMock";
import { notify } from "components/ToastNotification";
import {
  DiskEncryptionActionRequired,
  IBitLockerPINRequest,
  IDUPDetails,
  IOSSettings,
} from "interfaces/host";
import diskEncryptionAPI from "services/entities/disk_encryption";

import BitLockerPinModal, {
  POLL_INTERVAL_MS,
  POLL_TIMEOUT_MS,
} from "./BitLockerPinModal";

jest.mock("services/entities/disk_encryption", () => ({
  __esModule: true,
  default: { submitBitLockerPIN: jest.fn() },
}));

jest.mock("components/ToastNotification", () => ({
  notify: { success: jest.fn(), error: jest.fn() },
}));

const submitBitLockerPIN = diskEncryptionAPI.submitBitLockerPIN as jest.Mock;

/** The modal reads only this one path off the device response. */
const deviceDetails = (
  diskEncryption: IOSSettings["disk_encryption"]
): IDUPDetails => {
  const host = createMockHost();
  return ({
    host: {
      ...host,
      mdm: {
        ...host.mdm,
        os_settings: { certificates: [], disk_encryption: diskEncryption },
      },
    },
  } as unknown) as IDUPDetails;
};

/** A poll answering with this PIN request. Everything else is what a host still asking for a PIN reports. */
const pollReturning = (
  pinRequest?: IBitLockerPINRequest,
  actionRequired: DiskEncryptionActionRequired = "create_pin"
) =>
  jest.fn().mockResolvedValue(
    deviceDetails({
      status: "action_required",
      detail: "",
      action_required: actionRequired,
      pin_request: pinRequest,
    })
  );

const renderModal = (onPollHost = jest.fn(), onExit = jest.fn()) => {
  const user = userEvent.setup({ advanceTimers: jest.advanceTimersByTime });
  render(
    <BitLockerPinModal
      deviceAuthToken="token"
      onPollHost={onPollHost}
      onExit={onExit}
    />
  );
  return { user, onPollHost, onExit };
};

const submitPIN = async (
  user: ReturnType<typeof userEvent.setup>,
  pin: string,
  confirmPin = pin
) => {
  await user.type(screen.getByLabelText("BitLocker PIN"), pin);
  await user.type(screen.getByLabelText("Confirm PIN"), confirmPin);
  await user.click(screen.getByRole("button", { name: "Save" }));
};

/** Settles the submit request, then runs the modal's wait forward. */
const advanceWait = (ms = POLL_INTERVAL_MS) =>
  act(async () => {
    await Promise.resolve();
    await jest.advanceTimersByTimeAsync(ms);
  });

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

  it("closes with a success toast once the agent reports the PIN is set", async () => {
    const { user, onExit } = renderModal(
      pollReturning({ status: "set", error: "" })
    );

    // Spaces are part of a BitLocker PIN, so the modal must not trim them away.
    await submitPIN(user, " pin 1234 ");
    expect(submitBitLockerPIN).toHaveBeenCalledWith("token", " pin 1234 ");

    await advanceWait();

    expect(notify.success).toHaveBeenCalledWith("Successfully created PIN.");
    expect(onExit).toHaveBeenCalled();
  });

  it("keeps waiting while the agent has not answered", async () => {
    const { user, onExit } = renderModal(
      pollReturning({ status: "delivered", error: "" })
    );

    await submitPIN(user, "123456");
    await advanceWait();

    // The Save button carries the waiting state.
    expect(screen.getByText("Setting PIN...")).toBeVisible();
    expect(screen.queryByRole("button", { name: "Save" })).toBeNull();
    expect(notify.success).not.toHaveBeenCalled();
    expect(notify.error).not.toHaveBeenCalled();
    expect(onExit).not.toHaveBeenCalled();
  });

  it("does not claim success when the host stops asking for a PIN for another reason", async () => {
    // Protection going off drops action_required while the agent still holds the PIN.
    const { user, onExit } = renderModal(
      jest.fn().mockResolvedValue(
        deviceDetails({
          status: "action_required",
          detail: "",
          pin_request: { status: "delivered", error: "" },
        })
      )
    );

    await submitPIN(user, "123456");
    await advanceWait();

    expect(notify.success).not.toHaveBeenCalled();
    expect(onExit).not.toHaveBeenCalled();
    expect(screen.getByText("Setting PIN...")).toBeVisible();
  });

  it("closes without claiming failure when the agent has not answered in time", async () => {
    const { user, onExit } = renderModal(
      pollReturning({ status: "delivered", error: "" })
    );

    await submitPIN(user, "123456");
    await advanceWait(POLL_TIMEOUT_MS + POLL_INTERVAL_MS);

    // The server keeps the request collectable for far longer, so the PIN may still land.
    expect(notify.error).toHaveBeenCalledWith(
      "PIN submitted but Fleet is still working on it. You’ll see an update when this device responds."
    );
    expect(notify.success).not.toHaveBeenCalled();
    expect(onExit).toHaveBeenCalled();
  });

  it("stays open and reports the agent's reason when the PIN is refused", async () => {
    const { user, onExit } = renderModal(
      pollReturning({ status: "failed", error: "PIN already set." })
    );

    await submitPIN(user, "123456");
    await advanceWait();

    expect(notify.error).toHaveBeenCalledWith(
      "Couldn't set PIN. PIN already set. Try again or contact your IT admin."
    );
    expect(onExit).not.toHaveBeenCalled();
    // Save is offered again, so the button is out of its waiting state.
    expect(screen.getByRole("button", { name: "Save" })).toBeEnabled();
  });

  it("reports the agent's failure even once the host stops asking for a PIN", async () => {
    // A host can stop asking for a PIN for reasons unrelated to this submission.
    const { user, onExit } = renderModal(
      pollReturning({ status: "failed", error: "PIN already set" }, "restart")
    );

    await submitPIN(user, "123456");
    await advanceWait();

    expect(notify.error).toHaveBeenCalledWith(
      "Couldn't set PIN. PIN already set. Try again or contact your IT admin."
    );
    expect(notify.success).not.toHaveBeenCalled();
    expect(onExit).not.toHaveBeenCalled();
  });

  it("reports the server's reason for a rejected submission without waiting on the agent", async () => {
    const reason =
      "Fleet's agent on this host is too old to set a BitLocker PIN. Update fleetd and try again.";
    submitBitLockerPIN.mockRejectedValue({
      response: { data: { errors: [{ name: "base", reason }] } },
    });
    const { user, onPollHost, onExit } = renderModal();

    await submitPIN(user, "123456");
    await advanceWait();

    expect(notify.error).toHaveBeenCalledWith(
      `Couldn't set PIN. ${reason} Try again or contact your IT admin.`,
      expect.anything()
    );
    // A rejected submission is never in the agent's hands, so nothing is waited on.
    expect(onPollHost).not.toHaveBeenCalled();
    expect(onExit).not.toHaveBeenCalled();
  });
});
