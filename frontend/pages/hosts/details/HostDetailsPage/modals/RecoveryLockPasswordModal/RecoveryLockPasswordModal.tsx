import React, { useContext, useState } from "react";
import { useQuery } from "react-query";

import { getErrorReason } from "interfaces/errors";
import { IHostRecoveryLockPasswordResponse } from "interfaces/host";
import hostAPI from "services/entities/hosts";
import { NotificationContext } from "context/notification";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import InputFieldHiddenContent from "components/forms/fields/InputFieldHiddenContent";
import DataError from "components/DataError";
import Spinner from "components/Spinner";
import CustomLink from "components/CustomLink";
import Icon from "components/Icon";
import InfoBanner from "components/InfoBanner";
import TooltipWrapper from "components/TooltipWrapper";
import {
  DEFAULT_USE_QUERY_OPTIONS,
  LEARN_MORE_ABOUT_BASE_LINK,
} from "utilities/constants";
import { monthDayTimeFormat } from "utilities/date_format";

const baseClass = "recovery-lock-password-modal";

interface IRecoveryLockPasswordModalProps {
  hostId: number;
  canRotatePassword: boolean;
  onCancel: () => void;
}

const RecoveryLockPasswordModal = ({
  hostId,
  canRotatePassword,
  onCancel,
}: IRecoveryLockPasswordModalProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const [isRotating, setIsRotating] = useState(false);

  const {
    data: recoveryLockData,
    error: recoveryLockPasswordError,
    isLoading,
  } = useQuery<
    IHostRecoveryLockPasswordResponse,
    unknown,
    IHostRecoveryLockPasswordResponse["recovery_lock_password"]
  >(
    ["hostRecoveryLockPassword", hostId],
    () => hostAPI.getRecoveryLockPassword(hostId),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      select: (data) => data.recovery_lock_password,
      // prevent caching this sensitive string
      cacheTime: 0,
    }
  );

  const onRotatePassword = async () => {
    setIsRotating(true);
    try {
      await hostAPI.rotateRecoveryLockPassword(hostId);
      renderFlash(
        "success",
        "Successfully sent request to rotate Recovery Lock password."
      );
      onCancel();
    } catch (e) {
      const msg = getErrorReason(e).includes("already in progress")
        ? "Recovery lock password rotation is already in progress for this host."
        : "Couldn't send request to rotate Recovery Lock password. Please try again.";

      renderFlash("error", msg);
    }
    setIsRotating(false);
  };

  const renderRotateButton = () => {
    if (canRotatePassword) {
      return (
        <Button
          variant="inverse"
          onClick={onRotatePassword}
          disabled={isRotating}
          className={`${baseClass}__rotate-button`}
        >
          <Icon name="refresh" />
          {isRotating ? "Rotating..." : "Rotate password"}
        </Button>
      );
    }

    return (
      <span className={`${baseClass}__rotate-button--disabled`}>
        <TooltipWrapper
          underline={false}
          showArrow
          position="bottom"
          tipContent="Only users with the maintainer role and above can rotate password."
        >
          <span className={`${baseClass}__rotate-button-content`}>
            <Icon name="refresh" />
            Rotate password
          </span>
        </TooltipWrapper>
      </span>
    );
  };

  return (
    <Modal
      title="Recovery Lock password"
      onExit={onCancel}
      onEnter={onCancel}
      className={baseClass}
    >
      {isLoading && <Spinner />}
      {recoveryLockPasswordError ? (
        <DataError />
      ) : (
        !isLoading && (
          <>
            <InputFieldHiddenContent value={recoveryLockData?.password ?? ""} />
            <p>
              Use this to unlock and regain access to the host if the end user
              forgets their local password.{" "}
              <CustomLink
                newTab
                url={`${LEARN_MORE_ABOUT_BASE_LINK}/startup-security-macos`}
                text="Learn more"
              />
            </p>
            {recoveryLockData?.auto_rotate_at && (
              <InfoBanner color="yellow">
                Password rotates automatically after{" "}
                {monthDayTimeFormat(recoveryLockData.auto_rotate_at)}.
              </InfoBanner>
            )}
            <div className="modal-cta-wrap">
              <Button onClick={onCancel}>Close</Button>
              {renderRotateButton()}
            </div>
          </>
        )
      )}
    </Modal>
  );
};

export default RecoveryLockPasswordModal;
