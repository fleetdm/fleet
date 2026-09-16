import { act, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import React from "react";

import createMockHost from "__mocks__/hostMock";
import { notify } from "components/ToastNotification";
import { IDUPDetails, IOSSettings } from "interfaces/host";
import diskEncryptionAPI from "services/entities/disk_encryption";

import BitLockerPinModal from "./BitLockerPinModal";

jest.mock("services/entities/disk_encryption", () => ({
  __esModule: true,
  default: { submitBitLockerPIN: jest.fn() },
}));

jest.mock("components/ToastNotification", () => ({
  notify: { success: jest.fn(), error: jest.fn() },
}));

const submitBitLockerPIN = diskEncryptionAPI.submitBitLockerPIN as jest.Mock;

const POLL_INTERVAL_MS = 3000;
const POLL_TIMEOUT_MS = 90000;

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

/** Flushes the request the modal is waiting on, so it reaches its next state. */
const settle = () =>
  act(async () => {
    await Promise.resolve();
  });

/** Runs the modal's wait through one poll. */
const advanceOnePoll = () =>
  act(async () => {
    await jest.advanceTimersByTimeAsync(POLL_INTERVAL_MS);
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
    const onPollHost = jest.fn().mockResolvedValue(
      deviceDetails({
        status: "action_required",
        detail: "",
        action_required: "create_pin",
        pin_request: { status: "set", error: "" },
      })
    );
    const { user, onExit } = renderModal(onPollHost);

    // Spaces are part of a BitLocker PIN, so the modal must not trim them away.
    await submitPIN(user, " pin 1234 ");
    expect(submitBitLockerPIN).toHaveBeenCalledWith("token", " pin 1234 ");

    await settle();
    await advanceOnePoll();

    expect(notify.success).toHaveBeenCalledWith("Successfully set PIN.");
    expect(onExit).toHaveBeenCalled();
  });

  it("keeps waiting while the agent has not answered", async () => {
    const onPollHost = jest.fn().mockResolvedValue(
      deviceDetails({
        status: "action_required",
        detail: "",
        action_required: "create_pin",
        pin_request: { status: "delivered", error: "" },
      })
    );
    const { user, onExit } = renderModal(onPollHost);

    await submitPIN(user, "123456");
    await settle();
    await advanceOnePoll();

    expect(screen.getByText("Setting your PIN...")).toBeVisible();
    expect(notify.success).not.toHaveBeenCalled();
    expect(notify.error).not.toHaveBeenCalled();
    expect(onExit).not.toHaveBeenCalled();
  });

  it("closes without claiming failure when the agent has not answered in time", async () => {
    const onPollHost = jest.fn().mockResolvedValue(
      deviceDetails({
        status: "action_required",
        detail: "",
        action_required: "create_pin",
        pin_request: { status: "delivered", error: "" },
      })
    );
    const { user, onExit } = renderModal(onPollHost);

    await submitPIN(user, "123456");
    await settle();
    await act(async () => {
      await jest.advanceTimersByTimeAsync(POLL_TIMEOUT_MS + POLL_INTERVAL_MS);
    });

    // The server keeps the request collectable for far longer, so the PIN may still land.
    expect(notify.error).toHaveBeenCalledWith(
      "PIN submitted but Fleet is still working on it. You’ll see an update when this device responds."
    );
    expect(notify.success).not.toHaveBeenCalled();
    expect(onExit).toHaveBeenCalled();
  });

  it("stays open and reports the agent's reason when the PIN is refused", async () => {
    const onPollHost = jest.fn().mockResolvedValue(
      deviceDetails({
        status: "action_required",
        detail: "",
        action_required: "create_pin",
        pin_request: { status: "failed", error: "PIN already set." },
      })
    );
    const { user, onExit } = renderModal(onPollHost);

    await submitPIN(user, "123456");
    await settle();
    await advanceOnePoll();

    expect(notify.error).toHaveBeenCalledWith(
      "Couldn't set PIN. PIN already set. Try again or contact your IT admin."
    );
    expect(onExit).not.toHaveBeenCalled();
    expect(screen.queryByText("Setting your PIN...")).toBeNull();
  });

  it("reports the server's reason for a rejected submission without waiting on the agent", async () => {
    submitBitLockerPIN.mockRejectedValue({
      response: {
        data: {
          errors: [
            {
              name: "base",
              reason:
                "Fleet's agent on this host is too old to set a BitLocker PIN. Update fleetd and try again.",
            },
          ],
        },
      },
    });
    const onPollHost = jest.fn();
    const { user, onExit } = renderModal(onPollHost);

    await submitPIN(user, "123456");
    await settle();

    expect(notify.error).toHaveBeenCalledWith(
      "Couldn't set PIN. Fleet's agent on this host is too old to set a BitLocker PIN. Update fleetd and try again. Try again or contact your IT admin.",
      expect.anything()
    );
    expect(onPollHost).not.toHaveBeenCalled();
    expect(onExit).not.toHaveBeenCalled();
  });

  it("reports the agent's failure even once the host stops asking for a PIN", async () => {
    const onPollHost = jest.fn().mockResolvedValue(
      deviceDetails({
        status: "action_required",
        detail: "",
        // A host can stop asking for a PIN for reasons unrelated to this submission.
        action_required: "restart",
        pin_request: { status: "failed", error: "PIN already set" },
      })
    );
    const { user, onExit } = renderModal(onPollHost);

    await submitPIN(user, "123456");
    await settle();
    await advanceOnePoll();

    expect(notify.error).toHaveBeenCalledWith(
      "Couldn't set PIN. PIN already set. Try again or contact your IT admin."
    );
    expect(notify.success).not.toHaveBeenCalled();
    expect(onExit).not.toHaveBeenCalled();
  });
});
