import React, { useState, useContext } from "react";

import DataError from "components/DataError";
import Button from "components/buttons/Button";
import Modal from "components/Modal";
import { NotificationContext } from "context/notification";

import mdmAPI from "services/entities/mdm";
import { isAndroid, isIPadOrIPhone } from "interfaces/platform";

interface IUnenrollMdmModalProps {
  hostId: number;
  hostPlatform: string;
  hostName: string;
  onClose: () => void;
}

const baseClass = "unenroll-mdm-modal";

const UnenrollMdmModal = ({
  hostId,
  hostPlatform,
  hostName,
  onClose,
}: IUnenrollMdmModalProps) => {
  const [requestState, setRequestState] = useState<
    undefined | "unenrolling" | "error"
  >(undefined);

  const { renderFlash } = useContext(NotificationContext);

  const submitUnenrollMdm = async () => {
    setRequestState("unenrolling");
    try {
      await mdmAPI.unenrollHostFromMdm(hostId, 5000);
      renderFlash(
        "success",
        <>
          MDM will be turned off for <b>{hostName}</b> next time this host
          checks in.
        </>
      );
    } catch (unenrollMdmError: unknown) {
      renderFlash(
        "error",
        <>
          Failed to turn off MDM for <b>{hostName}</b>.
        </>
      );
    }
    onClose();
  };

  const generateDescription = () => {
    if (isIPadOrIPhone(hostPlatform)) {
      return (
        <>
          <p>Settings configured by Fleet will be removed.</p>
          <p>
            To re-enroll, go to <b>Hosts &gt; Add hosts &gt; iOS/iPadOS</b> and
            share the link with end user.
          </p>
        </>
      );
    }
    if (isAndroid(hostPlatform)) {
      return (
        <>
          <p>Company data and OS settings (work profile) will be deleted.</p>
          <p>
            To re-enroll, go to <b>Hosts &gt; Add hosts &gt; Android</b> and
            share the link with end user.
          </p>
        </>
      );
    }
    return (
      <>
        <p>Settings configured by Fleet will be removed.</p>
        <p>
          To turn on MDM again, ask the device user to follow the{" "}
          <b>Turn on MDM</b> instructions on their <b>My device</b> page.
        </p>
      </>
    );
  };

  const renderModalContent = () => {
    if (requestState === "error") {
      return <DataError />;
    }

    const buttonText =
      isIPadOrIPhone(hostPlatform) || isAndroid(hostPlatform)
        ? "Unenroll"
        : "Turn off";

    return (
      <>
        <div className={`${baseClass}__description`}>
          {generateDescription()}
        </div>
        <div className="modal-cta-wrap">
          <Button
            type="submit"
            variant="alert"
            onClick={submitUnenrollMdm}
            isLoading={requestState === "unenrolling"}
          >
            {buttonText}
          </Button>
          <Button onClick={onClose} variant="inverse-alert">
            Cancel
          </Button>
        </div>
      </>
    );
  };

  const title =
    isIPadOrIPhone(hostPlatform) || isAndroid(hostPlatform)
      ? "Unenroll"
      : "Turn off MDM";

  return (
    <Modal title={title} onExit={onClose} className={baseClass} width="medium">
      {renderModalContent()}
    </Modal>
  );
};

export default UnenrollMdmModal;
