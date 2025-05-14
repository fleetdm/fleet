import React, { useState, useContext } from "react";

import DataError from "components/DataError";
import Button from "components/buttons/Button";
import Modal from "components/Modal";
import { NotificationContext } from "context/notification";

import mdmAPI from "services/entities/mdm";
import { isIPadOrIPhone } from "interfaces/platform";
import CustomLink from "components/CustomLink";

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

  const renderModalContent = () => {
    if (requestState === "error") {
      return <DataError />;
    }

    const turnOnMDMInstructions = isIPadOrIPhone(hostPlatform) ? (
      <>
        invite the end user to{" "}
        <CustomLink
          text="enroll a BYOD iPhone or iPad"
          url="https://fleetdm.com/guides/enroll-byod-ios-ipados-hosts"
          newTab
        />
      </>
    ) : (
      <>
        ask the device user to follow the <b>Turn on MDM</b> instructions on
        their <b>My device</b> page.
      </>
    );

    return (
      <>
        <p className={`${baseClass}__description`}>
          Settings configured by Fleet will be removed.
          <br />
          <br />
          To turn on MDM again, {turnOnMDMInstructions}
        </p>
        <div className="modal-cta-wrap">
          <Button
            type="submit"
            variant="alert"
            onClick={submitUnenrollMdm}
            isLoading={requestState === "unenrolling"}
          >
            Turn off
          </Button>
          <Button onClick={onClose} variant="inverse-alert">
            Cancel
          </Button>
        </div>
      </>
    );
  };

  return (
    <Modal
      title="Turn off MDM"
      onExit={onClose}
      className={baseClass}
      width="medium"
    >
      {renderModalContent()}
    </Modal>
  );
};

export default UnenrollMdmModal;
