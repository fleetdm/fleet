import React, { useContext, useState } from "react";
import { useQuery } from "react-query";

import { IHostManagedAccountPasswordResponse } from "interfaces/host";
import hostAPI from "services/entities/hosts";
import { NotificationContext } from "context/notification";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import InputFieldHiddenContent from "components/forms/fields/InputFieldHiddenContent";
import DataError from "components/DataError";
import Spinner from "components/Spinner";
import Icon from "components/Icon";
import InfoBanner from "components/InfoBanner";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import { monthDayTimeFormat } from "utilities/date_format";
import { getErrorReason } from "interfaces/errors";

const baseClass = "managed-account-modal";

interface IManagedAccountModalProps {
  hostId: number;
  // TODO(JM-43890): Figma dev note says "Hide option if not Admin or
  // maintainer role." We're hiding here per the design, but the analogous
  // RecoveryLockPasswordModal disables-with-tooltip in the same situation.
  // Following up to align the two patterns.
  canRotatePassword: boolean;
  onCancel: () => void;
  onRotate: () => void;
}

const ManagedAccountModal = ({
  hostId,
  canRotatePassword,
  onCancel,
  onRotate,
}: IManagedAccountModalProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const [isRotating, setIsRotating] = useState(false);
  const [justRotated, setJustRotated] = useState(false);

  const {
    data: managedAccountData,
    error: managedAccountError,
    isLoading,
    refetch: refetchManagedAccountPassword,
  } = useQuery<
    IHostManagedAccountPasswordResponse,
    unknown,
    IHostManagedAccountPasswordResponse["managed_account_password"]
  >(
    ["hostManagedAccountPassword", hostId],
    () => hostAPI.getManagedAccountPassword(hostId),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      select: (data) => data.managed_account_password,
      // prevent caching this sensitive string
      cacheTime: 0,
    }
  );

  const onRotatePassword = async () => {
    setIsRotating(true);
    try {
      await hostAPI.rotateManagedLocalAccountPassword(hostId);
      // Refetch the password so the modal shows the freshly-rotated value and
      // the act of viewing it sets a new auto_rotate_at on the row.
      await refetchManagedAccountPassword();
      setJustRotated(true);
      renderFlash(
        "success",
        "Successfully sent request to rotate managed local account password."
      );
      // Notify parent so it can refetch host details + activities.
      onRotate();
    } catch (e) {
      const msg = getErrorReason(e);
      renderFlash(
        "error",
        msg ||
          "Couldn't send request to rotate managed local account password. Please try again."
      );
    }
    setIsRotating(false);
  };

  const showPendingRotationBanner =
    justRotated || managedAccountData?.pending_rotation === true;
  const autoRotateAt = managedAccountData?.auto_rotate_at;

  return (
    <Modal title="Managed account" onExit={onCancel} className={baseClass}>
      {isLoading && <Spinner />}
      {managedAccountError ? (
        <DataError />
      ) : (
        !isLoading && (
          <>
            <div className={`${baseClass}__username`}>
              <span className={`${baseClass}__label`}>Username</span>
              <span className={`${baseClass}__value`}>_fleetadmin</span>
            </div>
            <InputFieldHiddenContent
              value={managedAccountData?.password ?? ""}
              name="Password"
            />
            {showPendingRotationBanner ? (
              <InfoBanner color="yellow">
                Password will rotate once the host acknowledges the request.
              </InfoBanner>
            ) : (
              autoRotateAt && (
                <InfoBanner color="yellow">
                  Password rotates automatically after{" "}
                  {monthDayTimeFormat(autoRotateAt)}.
                </InfoBanner>
              )
            )}
            <div className="modal-cta-wrap">
              <Button onClick={onCancel}>Close</Button>
              {canRotatePassword && (
                <Button
                  variant="inverse"
                  onClick={onRotatePassword}
                  disabled={isRotating}
                  className={`${baseClass}__rotate-button`}
                >
                  <Icon name="refresh" />
                  {isRotating ? "Rotating..." : "Rotate password"}
                </Button>
              )}
            </div>
          </>
        )
      )}
    </Modal>
  );
};

export default ManagedAccountModal;
