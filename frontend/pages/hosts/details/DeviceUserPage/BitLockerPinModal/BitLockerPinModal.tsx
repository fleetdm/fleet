import React, { useEffect, useId, useRef, useState } from "react";

import Button from "components/buttons/Button";
import InputField from "components/forms/fields/InputField";
import Modal from "components/Modal";
import ModalFooter from "components/ModalFooter";
import { notify } from "components/ToastNotification";
import useFormValidation, { IFormErrors } from "hooks/useFormValidation";
import { IDUPDetails, IOSSettings } from "interfaces/host";
import diskEncryptionAPI from "services/entities/disk_encryption";

const baseClass = "bit-locker-pin-modal";

/** Matches the server's rules in server/mdm/microsoft/bitlocker_pin.go. */
const PIN_MIN_LENGTH = 6;
const PIN_MAX_LENGTH = 20;
const PRINTABLE_ASCII = /^[ -~]+$/;

/** The agent checks in every 30 seconds, so a host that is awake answers well inside this. */
const POLL_INTERVAL_MS = 3000;
const POLL_TIMEOUT_MS = 90000;

const CONTACT_ADMIN = "Try again or contact your IT admin.";
const NO_RESPONSE_ERROR = "Fleet didn't hear back from this device.";

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
  /** Refetches the device's host details and resolves with the fresh response. */
  onPollHost: () => Promise<IDUPDetails | undefined>;
  onExit: () => void;
}

const BitLockerPinModal = ({
  deviceAuthToken,
  onPollHost,
  onExit,
}: IBitLockerPinModalProps): JSX.Element => {
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

  /** Waits for the agent to report on the PIN, resolving with the error to show, or nothing on success. */
  const waitForAgent = async (): Promise<string | undefined> => {
    const deadline = Date.now() + POLL_TIMEOUT_MS;
    while (Date.now() < deadline) {
      // eslint-disable-next-line no-await-in-loop
      await new Promise((resolve) => {
        setTimeout(resolve, POLL_INTERVAL_MS);
      });
      if (!isOpen.current) {
        return undefined;
      }

      let diskEncryption: IOSSettings["disk_encryption"] | undefined;
      try {
        // eslint-disable-next-line no-await-in-loop
        const details = await onPollHost();
        diskEncryption = details?.host.mdm.os_settings?.disk_encryption;
      } catch {
        // A blip on one poll says nothing about the PIN, so keep waiting.
      }

      // An osquery report can clear action_required before the agent's own report lands, and either one means it worked.
      if (
        diskEncryption &&
        (diskEncryption.pin_request?.status === "set" ||
          diskEncryption.action_required !== "create_pin")
      ) {
        return undefined;
      }
      if (diskEncryption?.pin_request?.status === "failed") {
        return diskEncryption.pin_request.error || NO_RESPONSE_ERROR;
      }
    }
    return NO_RESPONSE_ERROR;
  };

  const onValidSubmit = async ({ pin }: IBitLockerPinFormData) => {
    try {
      await diskEncryptionAPI.submitBitLockerPIN(deviceAuthToken, pin);
    } catch (e) {
      notify.error(`Couldn't set PIN. ${CONTACT_ADMIN}`, { response: e });
      return;
    }

    setIsWaitingForAgent(true);
    const agentError = await waitForAgent();
    if (!isOpen.current) {
      return;
    }
    setIsWaitingForAgent(false);

    if (agentError) {
      notify.error(`Couldn't set PIN. ${agentError} ${CONTACT_ADMIN}`);
      return;
    }
    notify.success("Successfully set PIN.");
    onExit();
  };

  const isDisabled = isSubmitting || isWaitingForAgent;

  return (
    <Modal title="Create PIN" onExit={onExit} className={baseClass}>
      <>
        <form id={formId} onSubmit={handleSubmit(onValidSubmit)}>
          <p>
            Set a BitLocker PIN to protect your data if this device is lost or
            stolen. You&apos;ll need to enter it each time your device starts
            up.
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
            helpText={`Must be ${PIN_MIN_LENGTH} to ${PIN_MAX_LENGTH} characters. Keep it somewhere safe. Fleet can't show this PIN to you or your IT team later.`}
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
          {isWaitingForAgent && (
            <p className={`${baseClass}__waiting`}>Setting your PIN...</p>
          )}
        </form>
        <ModalFooter
          primaryButtons={
            <>
              <Button
                onClick={onExit}
                variant="secondary"
                disabled={isDisabled}
              >
                Cancel
              </Button>
              <Button
                type="submit"
                formId={formId}
                isLoading={isDisabled}
                disabled={isDisabled}
              >
                Save
              </Button>
            </>
          }
        />
      </>
    </Modal>
  );
};

export default BitLockerPinModal;
