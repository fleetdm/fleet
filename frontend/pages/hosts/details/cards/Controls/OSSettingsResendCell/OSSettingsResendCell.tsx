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

import { IHostMdmProfileWithAddedStatus } from "../OSSettingsTableConfig";

const baseClass = "os-settings-resend-cell";

interface IActionButtonProps {
  isPending: boolean;
  className: string;
  text: string;
  pendingText: string;
  onClick: () => void;
}

const ActionButton = ({
  isPending,
  className,
  text,
  pendingText,
  onClick,
}: IActionButtonProps) => {
  return (
    <Button
      disabled={isPending}
      // The row opens the details modal; don't let an action click do that too.
      onClick={(evt: React.MouseEvent) => {
        evt.stopPropagation();
        onClick();
      }}
      variant="subdued"
      className={className}
      icon="refresh"
    >
      {isPending ? pendingText : text}
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
  /** Fade in on row hover. Set for the table cell, not the modal footer. */
  revealOnRowHover?: boolean;
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
  revealOnRowHover = false,
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
      onProfileResent();
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

  const actionClass = (
    modifier: string,
    pendingModifier: string,
    isPending: boolean
  ) =>
    classnames(`${baseClass}__${modifier}-button`, {
      [`${baseClass}__${pendingModifier}`]: isPending,
      // Keep an in-flight action visible so the user sees it working.
      "row-hover-button": revealOnRowHover && !isPending,
    });

  return (
    <div className={baseClass}>
      {showResendButton && (
        <ActionButton
          isPending={isResending}
          className={actionClass("resend", "resending", isResending)}
          text="Resend"
          pendingText="Resending..."
          onClick={onResendProfile}
        />
      )}
      {showRotateButton && (
        <ActionButton
          isPending={isRotating}
          className={actionClass("rotate", "rotating", isRotating)}
          text="Rotate"
          pendingText="Rotating..."
          onClick={onRotatePassword}
        />
      )}
      {showResendHostNameButton && (
        <ActionButton
          isPending={isResending}
          className={actionClass("resend", "resending", isResending)}
          text="Resend"
          pendingText="Resending..."
          onClick={onResendHostNameTemplate}
        />
      )}
    </div>
  );
};

export default OSSettingsResendCell;
