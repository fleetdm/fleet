import React, { useContext } from "react";

import hostAPI from "services/entities/hosts";
import { getErrorReason } from "interfaces/errors";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
import { NotificationContext } from "context/notification";
import { isAndroid } from "interfaces/platform";

const baseClass = "wipe-modal";

interface IWipeModalProps {
  id: number;
  hostName: string;
  hostPlatform: string;
  isWindowsHost: boolean;
  onSuccess: () => void;
  onClose: () => void;
}

const WipeModal = ({
  id,
  hostName,
  hostPlatform,
  isWindowsHost,
  onSuccess,
  onClose,
}: IWipeModalProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const [lockChecked, setLockChecked] = React.useState(false);
  const [isWiping, setIsWiping] = React.useState(false);
  const isAndroidHost = isAndroid(hostPlatform);

  const onWipe = async () => {
    setIsWiping(true);
    try {
      await hostAPI.wipeHost(id);
      onSuccess();
      // Android uses the Figma-specified copy; other platforms keep their existing message to
      // avoid regressing copy they were QA'd against.
      renderFlash(
        "success",
        isAndroidHost
          ? "Successfully sent request to wipe this host."
          : "Wiping host or will wipe when the host comes online."
      );
    } catch (e) {
      // Android: surface the backend error reason when available (e.g. "Wipe is not supported for
      // personally-owned Android hosts. Use Unenroll instead.") and fall back to the Figma copy
      // when the error has no extractable reason. Other platforms keep their existing behavior.
      const errorReason = getErrorReason(e);
      renderFlash(
        "error",
        isAndroidHost
          ? errorReason ||
              "Couldn't send request to wipe this host. Please try again."
          : errorReason
      );
    }
    onClose();
    setIsWiping(false);
  };

  return (
    <Modal className={baseClass} title="Wipe" onExit={onClose}>
      <div className={`${baseClass}__modal-content`}>
        <p>All content will be erased on this host.</p>
        {isWindowsHost && (
          <p>
            To use the host again, you will have to do a Windows reinstall from
            a USB drive.
          </p>
        )}
        <div className={`${baseClass}__confirm-message`}>
          <span>
            <b>Please check to confirm:</b>
          </span>
          <Checkbox
            wrapperClassName={`${baseClass}__wipe-checkbox`}
            value={lockChecked}
            onChange={(value: boolean) => setLockChecked(value)}
          >
            I wish to wipe <b>{hostName}</b>
          </Checkbox>
        </div>
      </div>

      <div className="modal-cta-wrap">
        <Button
          type="button"
          onClick={onWipe}
          variant="alert"
          className="delete-loading"
          disabled={!lockChecked}
          isLoading={isWiping}
        >
          Wipe
        </Button>
        <Button onClick={onClose} variant="inverse-alert">
          Cancel
        </Button>
      </div>
    </Modal>
  );
};

export default WipeModal;
