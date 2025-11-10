import React from "react";

import {
  IHostSoftware,
  SoftwareInstallUninstallStatus,
} from "interfaces/software";
import Button from "components/buttons/Button";
import Modal from "components/Modal";
import IconStatusMessage from "components/IconStatusMessage";
import ModalFooter from "components/ModalFooter";
import InventoryVersions from "pages/hosts/details/components/InventoryVersions";

const baseClass = "software-update-modal";

// Status message render helper
interface IStatusMessageProps {
  hostDisplayName: string;
  isDeviceUser: boolean;
  softwareStatus: SoftwareInstallUninstallStatus | null;
  softwareName: string;
  installerName: string;
  installerVersion?: string;
}
const StatusMessage = ({
  hostDisplayName,
  isDeviceUser,
  softwareStatus,
  softwareName,
  installerName,
  installerVersion,
}: IStatusMessageProps) => {
  const renderMessage = () => {
    if (softwareStatus === "pending_install") {
      return (
        <>
          Fleet is updating or will update <b>{softwareName}</b>
          {installerName && ` (${installerName})`} on <b>{hostDisplayName}</b>{" "}
          when it comes online.
        </>
      );
    }
    if (isDeviceUser) {
      return (
        <>
          Update <b>{softwareName}</b> to {installerVersion}.
        </>
      );
    }
    return (
      <>
        New version of <b>{softwareName}</b>
        {installerVersion && ` (${installerVersion})`} is available. Update the
        current version on <b>{hostDisplayName}</b>.
      </>
    );
  };

  return (
    <IconStatusMessage
      className={`${baseClass}__status-message`}
      iconName={
        softwareStatus === "pending_install"
          ? "pending-outline"
          : "error-outline"
      }
      iconColor="ui-fleet-black-50"
      message={<span>{renderMessage()}</span>}
    />
  );
};

interface ISoftwareUpdateModalProps {
  hostDisplayName: string;
  software: IHostSoftware;
  onExit: () => void;
  isDeviceUser?: boolean;
  /** Currently API for updating is the same as installing */
  onUpdate: (id: number) => void;
}

const SoftwareUpdateModal = ({
  hostDisplayName,
  software,
  isDeviceUser = false,
  onExit,
  onUpdate,
}: ISoftwareUpdateModalProps) => {
  const {
    id,
    status,
    name,
    display_name,
    installed_versions,
    software_package,
    app_store_app,
  } = software;
  const installerName =
    software_package?.display_name || software_package?.name || "";
  const installerVersion = software_package?.version || app_store_app?.version;

  const onClickUpdate = () => {
    onUpdate(id);
    onExit();
  };

  const hasCurrentVersions =
    installed_versions && installed_versions.length > 0;
  const showCurrentVersions =
    hasCurrentVersions && software.status !== "pending_install";

  return (
    <Modal title="Update details" className={baseClass} onExit={onExit}>
      <>
        <div className={`${baseClass}__modal-content`}>
          <StatusMessage
            hostDisplayName={hostDisplayName}
            isDeviceUser={isDeviceUser}
            softwareStatus={status}
            softwareName={display_name || name}
            installerName={installerName}
            installerVersion={installerVersion}
          />
          {showCurrentVersions && <InventoryVersions hostSoftware={software} />}
        </div>
        <ModalFooter
          primaryButtons={
            status === "pending_install" ? (
              <Button type="submit" onClick={onExit}>
                Done
              </Button>
            ) : (
              <>
                <Button variant="inverse" onClick={onExit}>
                  Cancel
                </Button>
                <Button type="submit" onClick={onClickUpdate}>
                  Update
                </Button>
              </>
            )
          }
        />
      </>
    </Modal>
  );
};

export default SoftwareUpdateModal;
