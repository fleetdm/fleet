import React from "react";

import { ITeamSummary } from "interfaces/team";
import DataError from "components/DataError";
import Modal from "components/Modal";
import Spinner from "components/Spinner";

import PlatformWrapper from "./PlatformWrapper/PlatformWrapper";
import DownloadInstallers from "./DownloadInstallers/DownloadInstallers";

const baseClass = "add-hosts-modal";

interface IAddHostsModal {
  currentTeam?: ITeamSummary;
  enrollSecret?: string;
  isLoading: boolean;
  isSandboxMode?: boolean;
  onCancel: () => void;
}

const AddHostsModal = ({
  currentTeam,
  enrollSecret,
  isLoading,
  isSandboxMode,
  onCancel,
}: IAddHostsModal): JSX.Element => {
  const renderModalContent = () => {
    if (isLoading) {
      return <Spinner />;
    }
    if (!enrollSecret) {
      return <DataError />;
    }

    // TODO: Currently, prepacked installers in Fleet Sandbox use the global enroll secret,
    // and Fleet Sandbox runs Fleet Free so the currentTeam check here is an
    // additional precaution/reminder to revisit this in connection with future changes.
    // See https://github.com/fleetdm/fleet/issues/4970#issuecomment-1187679407.
    return isSandboxMode && !currentTeam ? (
      <DownloadInstallers onCancel={onCancel} enrollSecret={enrollSecret} />
    ) : (
      <PlatformWrapper onCancel={onCancel} enrollSecret={enrollSecret} />
    );
  };

  return (
    <Modal onExit={onCancel} title={"Add hosts"} className={baseClass}>
      {renderModalContent()}
    </Modal>
  );
};

export default AddHostsModal;
