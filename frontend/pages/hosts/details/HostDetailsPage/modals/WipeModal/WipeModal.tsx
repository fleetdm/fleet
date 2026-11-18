import React from "react";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import Checkbox from "components/forms/fields/Checkbox";
import Modal from "components/Modal";
import { notify } from "components/ToastNotification";
import { getErrorReason } from "interfaces/errors";
import { isAndroid, isAppleDevice } from "interfaces/platform";
import hostAPI from "services/entities/hosts";

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
  const isAppleHost = isAppleDevice(hostPlatform);

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
        {isWindowsHost && (
          <>
            <p>
              Wipe lost or stolen hosts. To use the host again, you will need to
              reinstall the OS.
            </p>
            <p>
              To repurpose the host for another user, use{" "}
              <b>Actions &gt; Reset</b>.
            </p>
          </>
        )}
        {isLinuxHost && (
          <>
            <p>
              Wipe lost or stolen hosts. To repurpose the host for another end
              user, you will need to reinstall the OS.
            </p>
            <p>
              This will run a script to erase all data from this host.{" "}
              <CustomLink
                url="https://fleetdm.com/learn-more-about/linux-wipe"
                text="Learn more"
                newTab
              />
            </p>
          </>
        )}
        {(isAppleHost || isAndroidHost) && (
          <p>
            Repurpose the host for another end user, or wipe lost or stolen
            hosts.
          </p>
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
        <Button onClick={onClose} variant="secondary">
          Cancel
        </Button>
      </div>
    </Modal>
  );
};

export default WipeModal;
