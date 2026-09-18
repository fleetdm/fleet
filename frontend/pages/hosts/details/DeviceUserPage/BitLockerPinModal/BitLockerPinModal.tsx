import React, { useEffect, useId, useRef, useState } from "react";

import Button from "components/buttons/Button";
import InputField from "components/forms/fields/InputField";
import Modal from "components/Modal";
import ModalFooter from "components/ModalFooter";
import { notify } from "components/ToastNotification";
import useFormValidation, { IFormErrors } from "hooks/useFormValidation";
import { getErrorReason } from "interfaces/errors";
import { IDUPDetails, IOSSettings } from "interfaces/host";
import diskEncryptionAPI from "services/entities/disk_encryption";

const baseClass = "bit-locker-pin-modal";

/** Matches the server's rules in server/mdm/microsoft/bitlocker_pin.go. */
const PIN_MIN_LENGTH = 6;
const PIN_MAX_LENGTH = 20;
const PRINTABLE_ASCII = /^[ -~]+$/;

/** The agent checks in every 30 seconds, so a host that is awake answers well inside this. */
export const POLL_INTERVAL_MS = 3000;
export const POLL_TIMEOUT_MS = 90000;

const CONTACT_ADMIN = "Try again or contact your IT admin.";

/** The server keeps the request collectable for far longer than the modal waits, so a quiet 90 seconds is not a failure. */
const STILL_WORKING =
  "PIN submitted but Fleet is still working on it. You’ll see an update when this device responds.";

/** Fix inconsistent punctuation. */
const asSentence = (reason: string) =>
  /[.!?]$/.test(reason) ? reason : `${reason}.`;

const couldNotSetPIN = (reason?: string) =>
  `Couldn't set PIN. ${reason ? `${asSentence(reason)} ` : ""}${CONTACT_ADMIN}`;

interface IBitLockerPinFormData {
  pin: string;
  confirmPin: string;
}

const validateBitLockerPinForm = ({
  pin,
  confirmPin,
}: IBitLockerPinFormData): IFormErrors => {
  const errors: IFormErrors = {};

  if (!pin) {
    errors.pin = "Enter a PIN";
  } else if (pin.length < PIN_MIN_LENGTH || pin.length > PIN_MAX_LENGTH) {
    errors.pin = `Use ${PIN_MIN_LENGTH} to ${PIN_MAX_LENGTH} characters`;
  } else if (!PRINTABLE_ASCII.test(pin)) {
    errors.pin =
      "Use only letters, numbers, spaces, and symbols from a US English keyboard";
  }

  if (!errors.pin && confirmPin !== pin) {
    errors.confirmPin = "PINs must match";
  }

  return errors;
};

type PINOutcome =
  | { status: "set" }
  | { status: "failed"; error: string }
  /** The modal stopped waiting. The agent can still collect and apply the PIN. */
  | { status: "waiting" };

interface IBitLockerPinModalProps {
  deviceAuthToken: string;
  /** Refetches the device's host details and resolves with the fresh response. */
  onPollHost: () => Promise<IDUPDetails | undefined>;
  onExit: () => void;
}

const BitLockerPinModal = ({
  deviceAuthToken,
  onPollHost,
  onExit,
}: IBitLockerPinModalProps) => {
  // The submit button lives in a ModalFooter outside the <form>, so it reaches the form's onSubmit through this id.
  const formId = useId();

  // Polling outlives a render, and it must stop when the end user closes the modal mid-wait.
  const isOpen = useRef(true);
  useEffect(() => {
    return () => {
      isOpen.current = false;
    };
  }, []);

  const [isWaitingForAgent, setIsWaitingForAgent] = useState(false);

  const {
    formData,
    setField,
    getError,
    clearFieldError,
    validateField,
    handleSubmit,
    isSubmitting,
  } = useFormValidation<IBitLockerPinFormData>({
    initialFormData: { pin: "", confirmPin: "" },
    validate: validateBitLockerPinForm,
    // A PIN's leading and trailing spaces are part of the credential.
    skipTrim: ["pin", "confirmPin"],
  });

  /** Waits for the agent to report on the PIN. "waiting" means the modal gave up first, not that the PIN was refused. */
  const waitForAgent = async (): Promise<PINOutcome> => {
    const deadline = Date.now() + POLL_TIMEOUT_MS;
    while (Date.now() < deadline) {
      // eslint-disable-next-line no-await-in-loop
      await new Promise((resolve) => {
        setTimeout(resolve, POLL_INTERVAL_MS);
      });
      if (!isOpen.current) {
        return { status: "waiting" };
      }

      let diskEncryption: IOSSettings["disk_encryption"] | undefined;
      try {
        // eslint-disable-next-line no-await-in-loop
        const details = await onPollHost();
        diskEncryption = details?.host.mdm.os_settings?.disk_encryption;
      } catch {
        // A blip on one poll says nothing about the PIN, so keep waiting.
      }

      // The agent's own report is checked first: a host can stop asking for a PIN for reasons unrelated to this
      // submission.
      if (diskEncryption?.pin_request?.status === "failed") {
        return { status: "failed", error: diskEncryption.pin_request.error };
      }
      // Success is the agent's report, or osquery already seeing the PIN.
      if (
        diskEncryption &&
        (diskEncryption.pin_request?.status === "set" ||
          diskEncryption.status === "verified" ||
          diskEncryption.status === "verifying")
      ) {
        return { status: "set" };
      }
    }
    return { status: "waiting" };
  };

  const onValidSubmit = async ({ pin }: IBitLockerPinFormData) => {
    try {
      await diskEncryptionAPI.submitBitLockerPIN(deviceAuthToken, pin);
    } catch (e) {
      // The server's reason is the actionable part for an expected rejection, such as a fleetd too old to be handed a PIN.
      notify.error(couldNotSetPIN(getErrorReason(e)), { response: e });
      return;
    }

    setIsWaitingForAgent(true);
    const outcome = await waitForAgent();
    if (!isOpen.current) {
      return;
    }
    setIsWaitingForAgent(false);

    if (outcome.status === "failed") {
      notify.error(couldNotSetPIN(outcome.error));
      return;
    }
    if (outcome.status === "set") {
      notify.success("Successfully created PIN.");
    } else {
      // Leaving the form open would invite a second PIN that supersedes the one the agent is still collecting.
      notify.error(STILL_WORKING);
    }
    onExit();
  };

  const isDisabled = isSubmitting || isWaitingForAgent;

  return (
    <Modal title="Create PIN" onExit={onExit} className={baseClass}>
      <form id={formId} onSubmit={handleSubmit(onValidSubmit)}>
        <p>
          Set a BitLocker PIN to protect your data if this device is lost or
          stolen. You&apos;ll need to enter it each time your device starts up.
        </p>
        <InputField
          label="BitLocker PIN"
          name="pin"
          type="password"
          value={formData.pin}
          error={getError("pin")}
          onChange={(value: string) => setField("pin", value)}
          onFocus={() => clearFieldError("pin")}
          onBlur={() => validateField("pin")}
          disabled={isDisabled}
          enableShowSecret
          blockAutoComplete
          autofocus
          helpText={`Must be ${PIN_MIN_LENGTH}–${PIN_MAX_LENGTH} characters. Keep it somewhere safe. This PIN isn't kept by Fleet or your IT team.`}
        />
        <InputField
          label="Confirm PIN"
          name="confirmPin"
          type="password"
          value={formData.confirmPin}
          error={getError("confirmPin")}
          onChange={(value: string) => setField("confirmPin", value)}
          onFocus={() => clearFieldError("confirmPin")}
          onBlur={() => validateField("confirmPin")}
          disabled={isDisabled}
          enableShowSecret
          blockAutoComplete
        />
      </form>
      <ModalFooter
        primaryButtons={
          <>
            <Button onClick={onExit} variant="secondary" disabled={isDisabled}>
              Cancel
            </Button>
            <Button
              type="submit"
              formId={formId}
              isLoading={isDisabled}
              loadingText="Setting PIN..."
              disabled={isDisabled}
            >
              Save
            </Button>
          </>
        }
      />
    </Modal>
  );
};

export default BitLockerPinModal;
