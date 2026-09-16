import React, { useContext } from "react";
import { useQuery } from "react-query";

import Button from "components/buttons/Button";
import Modal from "components/Modal";
import Spinner from "components/Spinner";
import { AppContext } from "context/app";
import configAPI from "services/entities/config";

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
        <>
          <p>You have no enroll secrets.</p>
          <p>
            New hosts will not enroll until an enroll secret is added to{" "}
            <b>{teamDisplayName}</b>.
          </p>
          {openEnrollSecretModal && (
            <div className="modal-cta-wrap">
              <Button onClick={onManageEnrollSecretsClick}>
                Add enroll secret
              </Button>
            </div>
          )}
        </>
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
