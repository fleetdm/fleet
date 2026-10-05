import React, { useContext } from "react";
import { useQuery } from "react-query";

import Button from "components/buttons/Button";
import Modal from "components/Modal";
import Spinner from "components/Spinner";
import { AppContext } from "context/app";
import configAPI from "services/entities/config";

import PlatformWrapper from "./PlatformWrapper/PlatformWrapper";

const baseClass = "add-hosts-modal";

// The hosts that can still enroll without an enroll secret, by which one-time enroll secrets are on.
export const APPLE_AUTO_ENROLLING_HOSTS =
  "Apple hosts that automatically enroll via Automated Device Enrollment (ADE)";
export const WINDOWS_AUTO_ENROLLING_HOSTS =
  "Windows hosts that automatically enroll via Microsoft Entra ID or Autopilot";
export const APPLE_AND_WINDOWS_AUTO_ENROLLING_HOSTS =
  "hosts that automatically enroll via Apple's Automated Device Enrollment (ADE), Microsoft Entra ID, or Autopilot";

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
      // Hosts that get a one-time enroll secret from Fleet's MDM can still enroll without one.
      const appleOneTimeSecrets = !!config?.auth
        ?.mdm_apple_one_time_enroll_secrets;
      const windowsOneTimeSecrets = !!config?.auth
        ?.mdm_windows_one_time_enroll_secrets;
      let autoEnrollingHosts = "";
      if (appleOneTimeSecrets && windowsOneTimeSecrets) {
        autoEnrollingHosts = APPLE_AND_WINDOWS_AUTO_ENROLLING_HOSTS;
      } else if (appleOneTimeSecrets) {
        autoEnrollingHosts = APPLE_AUTO_ENROLLING_HOSTS;
      } else if (windowsOneTimeSecrets) {
        autoEnrollingHosts = WINDOWS_AUTO_ENROLLING_HOSTS;
      }
      return (
        <>
          <p>You have no enroll secrets.</p>
          <p>
            {autoEnrollingHosts ? (
              <>
                Only {autoEnrollingHosts} can enroll to <b>{teamDisplayName}</b>
                . Add an enroll secret to enroll other hosts.
              </>
            ) : (
              <>
                New hosts will not enroll until an enroll secret is added to{" "}
                <b>{teamDisplayName}</b>.
              </>
            )}
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
