import React from "react";

import hostAPI from "services/entities/hosts";
import { getErrorReason } from "interfaces/errors";

import { notify } from "components/ToastNotification";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
import CustomLink from "components/CustomLink";
import { isAndroid } from "interfaces/platform";

const baseClass = "wipe-modal";

interface IWipeModalProps {
  id: number;
  hostName: string;
  hostPlatform: string;
  isWindowsHost: boolean;
  isLinuxHost: boolean;
  onSuccess: () => void;
  onClose: () => void;
}

const WipeModal = ({
  id,
  hostName,
  hostPlatform,
  isWindowsHost,
  isLinuxHost,
  onSuccess,
  onClose,
}: IWipeModalProps) => {
  const [lockChecked, setLockChecked] = React.useState(false);
  const [isWiping, setIsWiping] = React.useState(false);
  const isAndroidHost = isAndroid(hostPlatform);

  const onWipe = async () => {
    setIsWiping(true);
    try {
      await hostAPI.wipeHost(id);
      onSuccess();
      notify.success(
        isAndroidHost
          ? "Successfully sent request to wipe this host."
          : "Wiping host or will wipe when the host comes online."
      );
    } catch (e) {
      const errorReason = getErrorReason(e);
      notify.error(
        isAndroidHost
          ? errorReason ||
              "Couldn't send request to wipe this host. Please try again."
          : errorReason,
        { response: e }
      );
    }
    onClose();
    setIsWiping(false);
  };

  return (
    <Modal className={baseClass} title="Wipe" onExit={onClose}>
      <div className={`${baseClass}__modal-content`}>
        {!isLinuxHost && <p>All content will be erased on this host.</p>}
        {isWindowsHost && (
          <p>
            To use the host again, you will have to do a Windows reinstall from
            a USB drive.
          </p>
        )}
        {isLinuxHost && (
          <>
            <p>
              This will run a script to erase content from this host.{" "}
              <CustomLink
                url="https://fleetdm.com/learn-more-about/linux-wipe"
                text="Learn more"
                newTab
              />{" "}
            </p>
            <p>To use the host again, you will have to do an OS reinstall.</p>
          </>
        )}
        <div className={`${baseClass}__confirm-message`}>
          <span>
            <b>Confirm:</b>
          </span>
          <Checkbox
            wrapperClassName={`${baseClass}__wipe-checkbox`}
            value={lockChecked}
            onChange={(value: boolean) => setLockChecked(value)}
            variant="danger"
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
