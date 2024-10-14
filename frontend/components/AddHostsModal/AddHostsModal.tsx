import React, { useContext } from "react";
import { useQuery } from "react-query";
import configAPI from "services/entities/config";
import { AppContext } from "context/app";

import Button from "components/buttons/Button";
import DataError from "components/DataError";
import Modal from "components/Modal";
import Spinner from "components/Spinner";

import PlatformWrapper from "./PlatformWrapper/PlatformWrapper";

const baseClass = "add-hosts-modal";

interface IAddHostsModal {
  currentTeamName?: string;
  enrollSecret?: string;
  isAnyTeamSelected: boolean;
  isLoading: boolean;
  onCancel: () => void;
  openEnrollSecretModal?: () => void;
}

const AddHostsModal = ({
  currentTeamName,
  enrollSecret,
  isAnyTeamSelected,
  isLoading,
  onCancel,
  openEnrollSecretModal,
}: IAddHostsModal): JSX.Element => {
  const { isPreviewMode, config } = useContext(AppContext);
  const teamDisplayName = (isAnyTeamSelected && currentTeamName) || "Fleet";

  const {
    data: certificate,
    error: fetchCertificateError,
    isFetching: isFetchingCertificate,
  } = useQuery<string, Error>(
    ["certificate"],
    () => configAPI.loadCertificate(),
    {
      enabled: !isPreviewMode,
      refetchOnWindowFocus: false,
    }
  );

  const onManageEnrollSecretsClick = () => {
    onCancel();
    openEnrollSecretModal && openEnrollSecretModal();
  };

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

    return (
      <PlatformWrapper
        onCancel={onCancel}
        enrollSecret={enrollSecret}
        certificate={certificate}
        isFetchingCertificate={isFetchingCertificate}
        fetchCertificateError={fetchCertificateError}
        config={config}
      />
    );
  };

  return (
    <Modal
      onExit={onCancel}
      title="Add hosts"
      className={baseClass}
      width="large"
    >
      {renderModalContent()}
    </Modal>
  );
};

export default AddHostsModal;
