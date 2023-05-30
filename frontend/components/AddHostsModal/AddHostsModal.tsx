import React from "react";

import Button from "components/buttons/Button";
import DataError from "components/DataError";
import Modal from "components/Modal";
import Spinner from "components/Spinner";

import PlatformWrapper from "./PlatformWrapper/PlatformWrapper";
import DownloadInstallers from "./DownloadInstallers/DownloadInstallers";

const baseClass = "add-hosts-modal";

interface IAddHostsModal {
  currentTeamName?: string;
  enrollSecret?: string;
  isAnyTeamSelected: boolean;
  isLoading: boolean;
  isSandboxMode?: boolean;
  onCancel: () => void;
  openEnrollSecretModal?: () => void;
}

const AddHostsModal = ({
  currentTeamName,
  enrollSecret,
  isAnyTeamSelected,
  isLoading,
  isSandboxMode,
  onCancel,
  openEnrollSecretModal,
}: IAddHostsModal): JSX.Element => {
  const teamDisplayName = (isAnyTeamSelected && currentTeamName) || "Fleet";

  const onManageEnrollSecretsClick = () => {
    onCancel();
    openEnrollSecretModal && openEnrollSecretModal();
  };

  // TODO: Currently, prepacked installers in Fleet Sandbox use the global enroll secret,
  // and Fleet Sandbox runs Fleet Free so the currentTeam check here is an
  // additional precaution/reminder to revisit this in connection with future changes.
  // See https://github.com/fleetdm/fleet/issues/4970#issuecomment-1187679407.
  const shouldRenderDownloadInstallersContent =
    isSandboxMode && !isAnyTeamSelected;

  const renderModalContent = () => {
    if (isLoading) {
      return <Spinner />;
    }
    if (!enrollSecret) {
      return (
        <DataError>
          <span className="info__data">
            You have no enroll secrets.{" "}
            {openEnrollSecretModal ? (
              <Button onClick={onManageEnrollSecretsClick} variant="text-link">
                Manage enroll secrets
              </Button>
            ) : (
              "Manage enroll secrets"
            )}{" "}
            to enroll hosts to <b>{teamDisplayName}</b>.
          </span>
        </DataError>
      );
    }

    return shouldRenderDownloadInstallersContent ? (
      <DownloadInstallers onCancel={onCancel} enrollSecret={enrollSecret} />
    ) : (
      <PlatformWrapper onCancel={onCancel} enrollSecret={enrollSecret} />
    );
  };

  return (
    <Modal
      onExit={onCancel}
      title={"Add hosts"}
      className={baseClass}
      width="large"
    >
      {renderModalContent()}
    </Modal>
  );
};

export default AddHostsModal;
