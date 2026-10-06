import React, { useCallback, useState } from "react";

import Button from "components/buttons/Button";
import InfoBanner from "components/InfoBanner";
import Modal from "components/Modal";
import { notify } from "components/ToastNotification";
import { getErrorReason } from "interfaces/errors";
import softwareAPI from "services/entities/software";

const baseClass = "delete-software-modal";

const DELETE_SW_USED_BY_PATCH_POLICY_ERROR_MSG =
  "Couldn't delete. This software has a patch policy. Please remove the patch policy and try again.";
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
        Software <strong>will be uninstalled</strong> from hosts.
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
        started, they won&apos;t be canceled, and the results won&apos;t appear
        in Fleet.
      </p>
    </>
  );
};

interface IDeleteSoftwareModalProps {
  softwareId: number;
  teamId: number;
  /** Per-installer id on a multi-package title. When set, only this
   * specific package is deleted; otherwise the request deletes the legacy
   * single-package row (or VPP/FMA installer slot). */
  installerId?: number;
  /** Per-version id on a multi-version App Store app title (iOS/iPadOS/
   * Android). When set, only this specific version is deleted. */
  versionId?: number;
  /** True when the version being deleted is the only remaining version on
   * the title. Last-version delete also tears down the title's custom icon
   * and display name, so the modal surfaces that side-effect. */
  isLastVersion?: boolean;
  onExit: () => void;
  onSuccess: () => void;
  gitOpsModeEnabled?: boolean;
  isAppStoreApp?: boolean;
  isAndroidApp?: boolean;
  /** When true, the modal title reads "Delete package" instead of "Delete
   * software" and the title-level metadata warning is suppressed — we're
   * deleting one specific installer on a title that can hold several, not
   * the title itself. */
  canActivateMultiplePackages?: boolean;
}

const DeleteSoftwareModal = ({
  softwareId,
  teamId,
  installerId,
  versionId,
  isLastVersion = false,
  onExit,
  onSuccess,
  gitOpsModeEnabled,
  isAppStoreApp = false,
  isAndroidApp = false,
  canActivateMultiplePackages = false,
}: IDeleteSoftwareModalProps) => {
  const [isDeleting, setIsDeleting] = useState(false);
  const isVersionDelete = versionId !== undefined;

  const onDeleteSoftware = useCallback(async () => {
    setIsDeleting(true);
    try {
      await softwareAPI.deleteSoftwareInstaller(
        softwareId,
        teamId,
        installerId,
        versionId
      );
      notify.success("Successfully deleted software.");
      onSuccess();
    } catch (error) {
      const reason = getErrorReason(error);
      if (reason.includes("This software has a patch policy")) {
        notify.error(DELETE_SW_USED_BY_PATCH_POLICY_ERROR_MSG, {
          response: error,
        });
      } else if (reason.includes("Policy automation uses this software")) {
        notify.error(DELETE_SW_USED_BY_POLICY_ERROR_MSG, { response: error });
      } else if (reason.includes("This software is installed during")) {
        notify.error(DELETE_SW_INSTALLED_DURING_SETUP_ERROR_MSG, {
          response: error,
        });
      } else {
        notify.error("Couldn't delete. Please try again.", {
          response: error,
        });
      }
    }
    setIsDeleting(false);
    onExit();
  }, [softwareId, teamId, installerId, versionId, onSuccess, onExit]);

  const getModalTitle = (): string => {
    if (canActivateMultiplePackages) return "Delete package";
    if (isVersionDelete) return "Delete version";
    return "Delete software";
  };
  const modalTitle = getModalTitle();

  // Side-effect sentence on the title's custom icon and display name shows
  // when the delete tears down the whole title: full-title delete today (no
  // version/installer id) OR deleting the last remaining version.
  const showCustomIconSentence =
    !canActivateMultiplePackages && (!isVersionDelete || isLastVersion);

  return (
    <Modal
      className={baseClass}
      title={modalTitle}
      onExit={onExit}
      isContentDisabled={isDeleting}
    >
      {gitOpsModeEnabled && (
        <InfoBanner className={`${baseClass}__gitops-warning`}>
          You are currently in GitOps mode. If the package is defined in GitOps,
          it will reappear when GitOps runs.
        </InfoBanner>
      )}
      {getPlatformMessage(isAppStoreApp, isAndroidApp)}
      {showCustomIconSentence && (
        <p>Custom icon and display name will be deleted.</p>
      )}
      <div className="modal-cta-wrap">
        <Button
          variant="alert"
          onClick={onDeleteSoftware}
          isLoading={isDeleting}
        >
          Delete
        </Button>
        <Button variant="secondary" onClick={onExit}>
          Cancel
        </Button>
      </div>
    </Modal>
  );
};

export default DeleteSoftwareModal;
