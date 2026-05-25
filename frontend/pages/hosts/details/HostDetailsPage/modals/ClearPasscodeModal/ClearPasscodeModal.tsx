import React, { useContext } from "react";

import { NotificationContext } from "context/notification";
import hostAPI from "services/entities/hosts";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
import { isAndroid } from "interfaces/platform";
import { MdmEnrollmentStatus } from "interfaces/mdm";

const baseClass = "clear-passcode-modal";

interface IClearPasscodeModalProps {
  id: number;
  hostName: string;
  hostPlatform: string;
  hostMdmEnrollmentStatus?: MdmEnrollmentStatus | null;
  onExit: () => void;
}

const ClearPasscodeModal = ({
  id,
  hostName,
  hostPlatform,
  hostMdmEnrollmentStatus,
  onExit,
}: IClearPasscodeModalProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const [isClearingPasscode, setIsClearingPasscode] = React.useState(false);
  const [confirmChecked, setConfirmChecked] = React.useState(false);

  // Android: per Figma, the modal varies by ownership and requires a confirmation checkbox.
  // BYO clears only the work-profile passcode; COBO clears the device passcode. iOS / iPadOS
  // keeps its existing copy and no-checkbox flow unchanged.
  const isAndroidHost = isAndroid(hostPlatform);
  const isAndroidBYO =
    isAndroidHost && hostMdmEnrollmentStatus === "On (personal)";

  const onClearPasscode = async () => {
    setIsClearingPasscode(true);
    try {
      await hostAPI.clearPasscode(id);
      renderFlash(
        "success",
        isAndroidHost
          ? "Successfully sent request to clear the passcode for this host."
          : "Successfully sent request to clear passcode on this host."
      );
    } catch (e) {
      renderFlash(
        "error",
        isAndroidHost
          ? "Couldn't send request to clear the passcode for this host. Please try again."
          : "Couldn't send request to clear passcode on this host. Please try again."
      );
    } finally {
      onExit();
      setIsClearingPasscode(false);
    }
  };

  const renderBody = () => {
    if (isAndroidBYO) {
      return <p>This only clears the work profile passcode.</p>;
    }
    if (isAndroidHost) {
      return (
        <p>
          This will clear the host passcode. The user can unlock the device
          without entering a passcode.
        </p>
      );
    }
    return (
      <p>
        This will remove the current passcode and allow anyone with physical
        access to unlock the host.
      </p>
    );
  };

  const renderConfirmCheckbox = () => {
    if (!isAndroidHost) return null;
    return (
      <div className={`${baseClass}__confirm-message`}>
        <span>
          <b>Please check to confirm:</b>
        </span>
        <Checkbox
          wrapperClassName={`${baseClass}__clear-checkbox`}
          value={confirmChecked}
          onChange={(value: boolean) => setConfirmChecked(value)}
        >
          I wish to clear the passcode for <b>{hostName}</b>
        </Checkbox>
      </div>
    );
  };

  const isConfirmDisabled = isAndroidHost && !confirmChecked;

  return (
    <Modal className={baseClass} title="Clear passcode" onExit={onExit}>
      <div className={`${baseClass}__modal-content`}>
        {renderBody()}
        {renderConfirmCheckbox()}
      </div>

      <div className="modal-cta-wrap">
        <Button
          type="button"
          onClick={onClearPasscode}
          className="clear-passcode-loading"
          variant="alert"
          isLoading={isClearingPasscode}
          disabled={isConfirmDisabled}
        >
          Clear passcode
        </Button>
        <Button onClick={onExit} variant="inverse-alert">
          Cancel
        </Button>
      </div>
    </Modal>
  );
};

export default ClearPasscodeModal;
