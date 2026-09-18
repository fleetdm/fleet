import React, { useEffect, useId, useState } from "react";

import Button from "components/buttons/Button";
import InputField from "components/forms/fields/InputField";
import Modal from "components/Modal";
import ModalFooter from "components/ModalFooter";
import { notify } from "components/ToastNotification";
import useFormValidation, { IFormErrors } from "hooks/useFormValidation";
import { getErrorReason } from "interfaces/errors";
import { IDeviceDiskEncryptionSetting } from "interfaces/host";
import diskEncryptionAPI from "services/entities/disk_encryption";

const baseClass = "bit-locker-pin-modal";

/** Matches the server's rules in server/mdm/microsoft/bitlocker_pin.go. */
const PIN_MIN_LENGTH = 6;
const PIN_MAX_LENGTH = 20;
const PRINTABLE_ASCII = /^[ -~]+$/;

/** How long the modal waits for the agent. It checks in every 30 seconds, so a host that is awake answers well
 * inside this. The page keeps polling afterwards, so giving up here only ends the wait on screen. */
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

interface IBitLockerPinModalProps {
  deviceAuthToken: string;
  /** Disk encryption from the page's host query, which polls while a submission is in the agent's hands. */
  diskEncryption?: IDeviceDiskEncryptionSetting;
  /** When that query last succeeded. The modal ignores anything fetched before its own submit, so a failure from an
   * earlier attempt is never read as the outcome of this one. */
  dataUpdatedAt: number;
  /** Refetches the host now, so the page sees the submission and starts polling for its outcome. */
  onSubmitted: () => void;
  onExit: () => void;
}

const BitLockerPinModal = ({
  deviceAuthToken,
  diskEncryption,
  dataUpdatedAt,
  onSubmitted,
  onExit,
}: IBitLockerPinModalProps) => {
  // The submit button lives in a ModalFooter outside the <form>, so it reaches the form's onSubmit through this id.
  const formId = useId();

  /** When the PIN was handed to Fleet, and the cutoff for data this modal will read. Null once the wait is over. */
  const [submittedAt, setSubmittedAt] = useState<number | null>(null);

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

  const onValidSubmit = async ({ pin }: IBitLockerPinFormData) => {
    try {
      await diskEncryptionAPI.submitBitLockerPIN(deviceAuthToken, pin);
    } catch (e) {
      // The server's reason is the actionable part for an expected rejection, such as a fleetd too old to be handed a PIN.
      notify.error(couldNotSetPIN(getErrorReason(e)), { response: e });
      return;
    }
    setSubmittedAt(Date.now());
    onSubmitted();
  };

  // Read the agent's answer out of the page's data. Only data fetched after the submit counts.
  useEffect(() => {
    if (submittedAt === null || dataUpdatedAt <= submittedAt) {
      return;
    }
    // The agent's own report is checked first: a host can stop asking for a PIN for reasons unrelated to this
    // submission.
    if (diskEncryption?.pin_request?.status === "failed") {
      setSubmittedAt(null);
      notify.error(couldNotSetPIN(diskEncryption.pin_request.error));
      return;
    }
    // Success is the agent's report, or osquery already seeing the PIN. Only verified and verifying require it.
    if (
      diskEncryption?.pin_request?.status === "set" ||
      diskEncryption?.status === "verified" ||
      diskEncryption?.status === "verifying"
    ) {
      setSubmittedAt(null);
      notify.success("Successfully created PIN.");
      onExit();
    }
  }, [submittedAt, dataUpdatedAt, diskEncryption, onExit]);

  // Stop waiting on screen after the deadline. The request stays collectable for far longer, so this is not a failure.
  useEffect(() => {
    if (submittedAt === null) {
      return undefined;
    }
    const timer = setTimeout(() => {
      setSubmittedAt(null);
      // Leaving the form open would invite a second PIN that supersedes the one the agent is still collecting.
      notify.error(STILL_WORKING);
      onExit();
    }, POLL_TIMEOUT_MS);
    return () => clearTimeout(timer);
  }, [submittedAt, onExit]);

  const isWaiting = submittedAt !== null;
  const isDisabled = isSubmitting || isWaiting;

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
