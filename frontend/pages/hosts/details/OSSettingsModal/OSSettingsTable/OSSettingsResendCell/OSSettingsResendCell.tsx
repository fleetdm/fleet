import React, { useState } from "react";
import classnames from "classnames";
import { noop } from "lodash";

import {
  HOST_NAME_SYNTHETIC_PROFILE_UUID,
  REC_LOCK_SYNTHETIC_PROFILE_UUID,
} from "pages/hosts/details/helpers";

import { FLEET_ANDROID_CERTIFICATE_TEMPLATE_PROFILE_ID } from "interfaces/mdm";
import { getErrorReason } from "interfaces/errors";

import { notify } from "components/ToastNotification";
import Button from "components/buttons/Button";
import Icon from "components/Icon";

import { IHostMdmProfileWithAddedStatus } from "../OSSettingsTableConfig";

const baseClass = "os-settings-resend-cell";

interface IResendButtonProps {
  isResending: boolean;
  onClick: (evt: React.MouseEvent<HTMLButtonElement, MouseEvent>) => void;
}

const ResendButton = ({ isResending, onClick }: IResendButtonProps) => {
  const classNames = classnames(`${baseClass}__resend-button`, "resend-link", {
    [`${baseClass}__resending`]: isResending,
  });

  const buttonText = isResending ? "Resending..." : "Resend";

  return (
    <Button
      disabled={isResending}
      onClick={onClick}
      variant="secondary"
      className={classNames}
      size="small"
    >
      <Icon name="refresh" size="small" />
      {buttonText}
    </Button>
  );
};

interface IRotateButtonProps {
  isRotating: boolean;
  onClick: () => void;
}

const RotateButton = ({ isRotating, onClick }: IRotateButtonProps) => {
  const classNames = classnames(`${baseClass}__rotate-button`, "rotate-link", {
    [`${baseClass}__rotating`]: isRotating,
  });

  const buttonText = isRotating ? "Rotating..." : "Rotate";

  return (
    <Button
      disabled={isRotating}
      onClick={onClick}
      variant="secondary"
      className={classNames}
      size="small"
    >
      <Icon name="refresh" size="small" />
      {buttonText}
    </Button>
  );
};

interface IOSSettingsResendCellProps {
  canResendProfiles: boolean;
  canRotateRecoveryLockPassword?: boolean;
  canResendHostNameTemplate?: boolean;
  profile: IHostMdmProfileWithAddedStatus;
  resendRequest: (profileUUID: string) => Promise<void>;
  resendCertificateRequest?: (certificateTemplateId: number) => Promise<void>;
  rotateRecoveryLockPassword?: () => Promise<void>;
  resendHostNameTemplate?: () => Promise<void>;
  onProfileResent?: () => void;
}

const OSSettingsResendCell = ({
  canResendProfiles,
  canRotateRecoveryLockPassword = false,
  canResendHostNameTemplate = false,
  profile,
  resendRequest,
  resendCertificateRequest,
  rotateRecoveryLockPassword,
  resendHostNameTemplate,
  onProfileResent = noop,
}: IOSSettingsResendCellProps) => {
  const [isResending, setIsResending] = useState(false);
  const [isRotating, setIsRotating] = useState(false);

  const isAndroidCertificate =
    profile.profile_uuid === FLEET_ANDROID_CERTIFICATE_TEMPLATE_PROFILE_ID;

  const onResendProfile = async () => {
    setIsResending(true);
    try {
      if (
        isAndroidCertificate &&
        resendCertificateRequest &&
        profile.certificate_template_id !== undefined
      ) {
        await resendCertificateRequest(profile.certificate_template_id);
        notify.success("Successfully sent request to resend certificate.");
        onProfileResent();
      } else if (!isAndroidCertificate) {
        await resendRequest(profile.profile_uuid);
        onProfileResent();
      }
    } catch (e) {
      notify.error("Couldn't resend. Please try again.", { response: e });
    }
    setIsResending(false);
  };

  const onRotatePassword = async () => {
    if (!rotateRecoveryLockPassword) return;
    setIsRotating(true);
    try {
      await rotateRecoveryLockPassword();
      notify.success(
        "Successfully sent request to rotate Recovery Lock password."
      );
    } catch (e) {
      const msg = getErrorReason(e).includes("already in progress")
        ? "Recovery lock password rotation is already in progress for this host."
        : "Couldn't send request to rotate Recovery Lock password. Please try again.";

      notify.error(msg, { response: e });
    }
    setIsRotating(false);
  };

  const onResendHostNameTemplate = async () => {
    if (!resendHostNameTemplate) return;
    setIsResending(true);
    try {
      await resendHostNameTemplate();
      onProfileResent();
    } catch (e) {
      notify.error("Couldn't resend. Please try again.", { response: e });
    }
    setIsResending(false);
  };

  const isFailed = profile.status === "failed";
  const isVerified = profile.status === "verified";
  const isRecoveryLockRow =
    profile.profile_uuid === REC_LOCK_SYNTHETIC_PROFILE_UUID;
  const isHostNameRow =
    profile.profile_uuid === HOST_NAME_SYNTHETIC_PROFILE_UUID;

  // The host name row is a synthetic row resent through its own endpoint, so it
  // must not go through the profile-resend path above.
  const showResendButton =
    canResendProfiles &&
    (isFailed || isVerified) &&
    !isRecoveryLockRow &&
    !isHostNameRow;
  const showRotateButton =
    canRotateRecoveryLockPassword && (isFailed || isVerified);
  // canResendHostNameTemplate is already pre-gated on the host name row by the
  // caller, mirroring how showRotateButton relies on canRotateRecoveryLockPassword.
  const showResendHostNameButton =
    canResendHostNameTemplate && (isFailed || isVerified);

  return (
    <div className={baseClass}>
      {showResendButton && (
        <ResendButton isResending={isResending} onClick={onResendProfile} />
      )}
      {showRotateButton && (
        <RotateButton isRotating={isRotating} onClick={onRotatePassword} />
      )}
      {showResendHostNameButton && (
        <ResendButton
          isResending={isResending}
          onClick={onResendHostNameTemplate}
        />
      )}
    </div>
  );
};

export default OSSettingsResendCell;
