import React, { useState, useContext } from "react";

import DataError from "components/DataError";
import Button from "components/buttons/Button";
import Modal from "components/Modal";
import { NotificationContext } from "context/notification";
import CustomLink from "components/CustomLink";

import mdmAPI from "services/entities/mdm";
import { isAndroid, isIPadOrIPhone } from "interfaces/platform";
import {
  isBYODAccountDrivenEnrollment,
  isBYODManualEnrollment,
  isBYODCompanyOwnedEnrollment,
  MdmEnrollmentStatus,
} from "interfaces/mdm";

const baseClass = "unenroll-mdm-modal";

interface IUnenrollMdmModalProps {
  hostId: number;
  hostPlatform: string;
  hostName: string;
  enrollmentStatus: MdmEnrollmentStatus | null;
  onClose: () => void;
}

const UnenrollMdmModal = ({
  hostId,
  hostPlatform,
  hostName,
  enrollmentStatus,
  onClose,
}: IUnenrollMdmModalProps) => {
  const [requestState, setRequestState] = useState<
    undefined | "unenrolling" | "error"
  >(undefined);

  const { renderFlash } = useContext(NotificationContext);

  const submitUnenrollMdm = async () => {
    setRequestState("unenrolling");
    try {
      if (isAndroid(hostPlatform)) {
        await mdmAPI.unenrollAndroidHostFromMdm(hostId, 5000);
      } else {
        await mdmAPI.unenrollHostFromMdm(hostId, 5000);
      }
      const successMessage =
        isIPadOrIPhone(hostPlatform) || isAndroid(hostPlatform) ? (
          <>
            <b>{hostName}</b> will be unenrolled next time this host checks in.
          </>
        ) : (
          <>
            MDM will be turned off for <b>{hostName}</b> next time this host
            checks in.
          </>
        );
      renderFlash("success", successMessage);
      onClose();
    } catch (unenrollMdmError: unknown) {
      const errorMessage =
        isIPadOrIPhone(hostPlatform) || isAndroid(hostPlatform) ? (
          "Couldn't unenroll. Please try again."
        ) : (
          <>
            Failed to turn off MDM for <b>{hostName}</b>. Please try again.
          </>
        );
      renderFlash("error", errorMessage);
    }
    setRequestState(undefined);
  };

  const generateIosOrIpadosDescription = () => {
    if (isBYODManualEnrollment(enrollmentStatus)) {
      return (
        <p>
          To re-enroll, invite the end user to{" "}
          <CustomLink
            text="enroll a BYOD iPhone or iPad"
            url="https://fleetdm.com/guides/enroll-byod-ios-ipados-hosts"
            newTab
          />
        </p>
      );
    } else if (isBYODAccountDrivenEnrollment(enrollmentStatus)) {
      return (
        <p>
          To re-enroll, ask your end user to navigate to{" "}
          <b>
            Settings &gt; General &gt; VPN &amp; Device Management &gt; Sign in
            to Work or School Account...
          </b>{" "}
          on their host and to log in with their work email.
        </p>
      );
    } else if (isBYODCompanyOwnedEnrollment(enrollmentStatus)) {
      return (
        <p>
          To re-enroll, make sure that the host is still in Apple Business
          Manager (ABM). The host will automatically enroll after it&apos;s
          reset.
        </p>
      );
    }
    return null;
  };

  const generateDescription = () => {
    if (isIPadOrIPhone(hostPlatform)) {
      return (
        <>
          <p>Settings configured by Fleet will be removed.</p>
          {generateIosOrIpadosDescription()}
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
    <Modal
      title={title}
      onExit={onClose}
      className={baseClass}
      width="medium"
      isContentDisabled={requestState === "unenrolling"}
    >
      {renderModalContent()}
    </Modal>
  );
};

export default UnenrollMdmModal;
