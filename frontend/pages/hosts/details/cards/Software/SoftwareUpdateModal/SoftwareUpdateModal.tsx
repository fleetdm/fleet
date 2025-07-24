import React from "react";

import { IHostSoftware, SoftwareInstallStatus } from "interfaces/software";

import Button from "components/buttons/Button";
import Modal from "components/Modal";
import Icon from "components/Icon";
import ModalFooter from "components/ModalFooter";

const baseClass = "software-update-modal";

interface IStatusMessageProps {
  hostDisplayName: string;
  isDeviceUser: boolean;
  softwareStatus: SoftwareInstallStatus | null;
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
          Fleet is updating or will update <b>{softwareName}</b>{" "}
          {installerName && `(${installerName})`} on <b>{hostDisplayName}</b>{" "}
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
        New version of <b>{softwareName}</b>{" "}
        {installerVersion && `(${installerVersion})`} is available. Update the
        current version on <b>{hostDisplayName}</b>.
      </>
    );
  };

  return (
    <div className={`${baseClass}__status-message`}>
      <Icon
        name={
          softwareStatus === "pending_install"
            ? "pending-outline"
            : "error-outline"
        }
        color="ui-fleet-black-50"
      />
      <span>{renderMessage()}</span>
    </div>
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
  const onClickUpdate = () => {
    onUpdate(software.id);
    onExit();
  };

  const renderStatus = () => {
    return (
      <StatusMessage
        hostDisplayName={hostDisplayName}
        isDeviceUser={isDeviceUser}
        softwareStatus={software.status}
        softwareName={software.name}
        installerName={software.software_package?.name || ""}
        installerVersion={
          software.software_package?.version || software.app_store_app?.version
        }
      />
    );
  };

  const renderCurrentVersions = () => {
    return <></>;
  };

  const renderFooter = () => {
    const renderPrimaryButtons = () => {
      if (software.status === "pending_install") {
        return (
          <Button type="submit" onClick={onExit}>
            Done
          </Button>
        );
      }

      return (
        <>
          <Button variant="inverse" onClick={onExit}>
            Cancel
          </Button>{" "}
          <Button type="submit" onClick={onClickUpdate}>
            Update
          </Button>
        </>
      );
    };

    return <ModalFooter primaryButtons={renderPrimaryButtons()} />;
  };

  return (
    <Modal title="Update details" className={baseClass} onExit={onExit}>
      <>
        {renderStatus()}
        {renderCurrentVersions()}
        {renderFooter()}
      </>
    </Modal>
  );
};

export default SoftwareUpdateModal;
