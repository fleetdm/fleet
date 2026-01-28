import React, { useCallback, useContext, useState } from "react";

import softwareAPI from "services/entities/software";
import { NotificationContext } from "context/notification";

import { getErrorReason } from "interfaces/errors";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import InfoBanner from "components/InfoBanner";

const baseClass = "delete-software-modal";

const DELETE_SW_USED_BY_POLICY_ERROR_MSG =
  "Couldn't delete. Policy automation uses this software. Please disable policy automation for this software and try again.";
const DELETE_SW_INSTALLED_DURING_SETUP_ERROR_MSG = (
  <>
    Couldn&apos;t delete. This software is installed during new host setup.
    Please remove software in <strong>Controls &gt; Setup experience</strong>{" "}
    and try again.
  </>
);

const getPlatformMessage = (isAppStoreApp: boolean, isAndroidApp: boolean) => {
  // Android apps do not have pending installs/uninstalls as they are initiated through setup experience or by user
  if (isAndroidApp) {
    return (
      <p>
        Currently, software won&apos;t be deleted from self-service (managed
        Google Play) and won&apos;t be uninstalled from the hosts.
      </p>
    );
  }

  // VPP apps pending installs/uninstalls commands are not cancelled (future story #25912) but results only show in activity feed, as software is removed from host's software library
  if (isAppStoreApp) {
    return (
      <>
        <p>
          Software <strong>won&apos;t be uninstalled</strong> from hosts.
        </p>
        <p>
          Pending or already started installs and uninstalls won&apos;t be
          canceled, and the results won&apos;t appear in Fleet.
        </p>
      </>
    );
  }

  return (
    <>
      <p>
        Software <strong>won&apos;t be uninstalled</strong> from hosts.
      </p>
      <p>
        Pending installs and uninstalls will be canceled. If they have already
        started, they won&apos; be canceled, and the results won&apos;t appear
        in Fleet.
      </p>
    </>
  );
};

interface IDeleteSoftwareModalProps {
  softwareId: number;
  teamId: number;
  onExit: () => void;
  onSuccess: () => void;
  gitOpsModeEnabled?: boolean;
  isAppStoreApp?: boolean;
  isAndroidApp?: boolean;
}

const DeleteSoftwareModal = ({
  softwareId,
  teamId,
  onExit,
  onSuccess,
  gitOpsModeEnabled,
  isAppStoreApp = false,
  isAndroidApp = false,
}: IDeleteSoftwareModalProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const [isDeleting, setIsDeleting] = useState(false);

  const onDeleteSoftware = useCallback(async () => {
    setIsDeleting(true);
    try {
      await softwareAPI.deleteSoftwareInstaller(softwareId, teamId);
      renderFlash("success", "Software deleted successfully!");
      onSuccess();
    } catch (error) {
      const reason = getErrorReason(error);
      if (reason.includes("Policy automation uses this software")) {
        renderFlash("error", DELETE_SW_USED_BY_POLICY_ERROR_MSG);
      } else if (reason.includes("This software is installed during")) {
        renderFlash("error", DELETE_SW_INSTALLED_DURING_SETUP_ERROR_MSG);
      } else {
        renderFlash("error", "Couldn't delete. Please try again.");
      }
    }
    setIsDeleting(false);
    onExit();
  }, [softwareId, teamId, renderFlash, onSuccess, onExit]);

  return (
    <Modal
      className={baseClass}
      title="Delete software"
      onExit={onExit}
      isContentDisabled={isDeleting}
    >
      <>
        {gitOpsModeEnabled && (
          <InfoBanner className={`${baseClass}__gitops-warning`}>
            You are currently in GitOps mode. If the package is defined in
            GitOps, it will reappear when GitOps runs.
          </InfoBanner>
        )}
        {getPlatformMessage(isAppStoreApp, isAndroidApp)}
        <p>Custom icon and display name will be deleted.</p>
        <div className="modal-cta-wrap">
          <Button
            variant="alert"
            onClick={onDeleteSoftware}
            isLoading={isDeleting}
          >
            Delete
          </Button>
          <Button variant="inverse-alert" onClick={onExit}>
            Cancel
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default DeleteSoftwareModal;
