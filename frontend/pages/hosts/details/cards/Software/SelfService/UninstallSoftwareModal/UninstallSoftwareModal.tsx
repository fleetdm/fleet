import React, { useCallback, useContext, useState } from "react";

import deviceUserAPI from "services/entities/device_user";
import { NotificationContext } from "context/notification";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";

const baseClass = "uninstall-software-modal";

interface IUninstallSoftwareModalProps {
  softwareId: number;
  softwareName?: string;
  softwareInstallerType?: string;
  version?: string;
  token: string;
  onExit: () => void;
  onSuccess: () => void;
}

const UninstallSoftwareModal = ({
  softwareId,
  softwareName,
  softwareInstallerType,
  version,
  token,
  onExit,
  onSuccess,
}: IUninstallSoftwareModalProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const [isUninstalling, setIsUninstalling] = useState(false);

  const onUninstallSoftware = useCallback(async () => {
    setIsUninstalling(true);
    try {
      await deviceUserAPI.uninstallSelfServiceSoftware(token, softwareId);
      onSuccess();
    } catch (error) {
      // We only show toast message to end user if API returns an error
      renderFlash("error", "Couldn't uninstall. Please try again.");
    }
    setIsUninstalling(false);
    onExit();
  }, [softwareId, renderFlash, onSuccess, onExit]);

  const displaySoftwareName = softwareName || "software";

  return (
    <Modal
      className={baseClass}
      title={`Uninstall ${displaySoftwareName}`}
      onExit={onExit}
      isContentDisabled={isUninstalling}
    >
      <>
        <p>
          Uninstalling this software will remove it and may remove{" "}
          {softwareName} data from your device. You can always reinstall it
          again later.
        </p>
        <div className="modal-cta-wrap">
          <Button
            variant="alert"
            onClick={onUninstallSoftware}
            isLoading={isUninstalling}
          >
            Uninstall
          </Button>
          <Button variant="inverse-alert" onClick={onExit}>
            Cancel
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default UninstallSoftwareModal;
