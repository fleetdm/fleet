import classnames from "classnames";
import { noop } from "lodash";
import React, { useState } from "react";

import Button from "components/buttons/Button";
import { IconNames } from "components/icons";
import { notify } from "components/ToastNotification";
import TooltipWrapper from "components/TooltipWrapper";
import { getErrorReason } from "interfaces/errors";
import { FLEET_ANDROID_CERTIFICATE_TEMPLATE_PROFILE_ID } from "interfaces/mdm";
import { getResendProfileErrorMessage } from "pages/hosts/details/cards/Controls/helpers";
import {
  HOST_NAME_SYNTHETIC_PROFILE_UUID,
  REC_LOCK_SYNTHETIC_PROFILE_UUID,
} from "pages/hosts/details/helpers";

import { IHostMdmProfileWithAddedStatus } from "../OSSettingsTableConfig";

const baseClass = "os-settings-actions-cell";

// Android config profiles (unlike certificates) are synced by the host
// checking in with Google periodically, similarly to Apple declaration (DDM)
// profiles, rather than Fleet pushing them — so they can't be resent on demand.
const ANDROID_PROFILE_NO_RESEND_TOOLTIP_MESSAGE =
  "Fleet can't resend this configuration profile. Android hosts check in for profiles periodically, rather than Fleet pushing them.";

const NO_RESEND_PERMISSION_TOOLTIP_MESSAGE =
  "You don't have permission to resend this profile.";

const PROFILE_NO_RESEND_TOOLTIP_MESSAGE =
  "Fleet can't resend this configuration profile until it's verified or failed.";

interface IActionButtonProps {
  className: string;
  text: string;
  icon: IconNames;
  onClick?: () => void;
  isPending?: boolean;
  pendingText?: string;
  disabled?: boolean;
  tooltip?: React.ReactNode;
  size?: "small" | "default";
}

const ActionButton = ({
  className,
  text,
  icon,
  onClick,
  isPending = false,
  pendingText,
  disabled = false,
  tooltip,
  size = "small",
}: IActionButtonProps) => (
  <TooltipWrapper
    tipContent={tooltip}
    disableTooltip={!tooltip}
    underline={false}
    // Actions sit at the table's right edge; a centered tooltip can collide
    // with it and flip sideways, so anchor the tooltip's right edge instead.
    position="top-end"
    showArrow
  >
    <Button
      variant="secondary"
      size={size}
      icon={icon}
      className={className}
      disabled={disabled || isPending}
      // The row opens the details modal; don't let an action click do that too.
      onClick={(evt: React.MouseEvent) => {
        evt.stopPropagation();
        onClick?.();
      }}
    >
      {isPending && pendingText ? pendingText : text}
    </Button>
  </TooltipWrapper>
);

interface IOSSettingsActionsCellProps {
  canResendProfiles: boolean;
  /** Also offer Resend while the profile is "verifying" (Fleetd configuration
   * profile with one-time enroll secrets). */
  canResendWhileVerifying?: boolean;
  canRotateRecoveryLockPassword?: boolean;
  canResendHostNameTemplate?: boolean;
  /** Shows a disabled "Resend" button with a tooltip explaining why, for
   * Android configuration profiles (which sync automatically and can't be
   * resent on demand). */
  showDisabledResendForAndroidProfile?: boolean;
  /** The profile could be resent, but not by this user. */
  lacksResendPermission?: boolean;
  profile: IHostMdmProfileWithAddedStatus;
  resendRequest: (profileUUID: string) => Promise<void>;
  resendCertificateRequest?: (certificateTemplateId: number) => Promise<void>;
  rotateRecoveryLockPassword?: () => Promise<void>;
  resendHostNameTemplate?: () => Promise<void>;
  onProfileResent?: () => void | Promise<unknown>;
  /** Offer Install/Uninstall on self-service (opt-in) profiles. */
  canManageSelfServiceProfiles?: boolean;
  onInstall?: (profile: IHostMdmProfileWithAddedStatus) => Promise<void>;
  /** Uninstall is confirmed in a modal owned by the caller. */
  onClickUninstall?: (profile: IHostMdmProfileWithAddedStatus) => void;
  /** An install/uninstall was sent and the host doesn't reflect it yet. */
  isActionRequested?: boolean;
  /** Full-size buttons to sit beside the details modal's footer buttons. */
  isInModal?: boolean;
}

const OSSettingsActionsCell = ({
  canResendProfiles,
  canResendWhileVerifying = false,
  canRotateRecoveryLockPassword = false,
  canResendHostNameTemplate = false,
  showDisabledResendForAndroidProfile = false,
  lacksResendPermission = false,
  profile,
  resendRequest,
  resendCertificateRequest,
  rotateRecoveryLockPassword,
  resendHostNameTemplate,
  onProfileResent = noop,
  canManageSelfServiceProfiles = false,
  onInstall,
  onClickUninstall,
  isActionRequested = false,
  isInModal = false,
}: IOSSettingsActionsCellProps) => {
  const buttonSize = isInModal ? "default" : "small";
  const [isResending, setIsResending] = useState(false);
  const [isRotating, setIsRotating] = useState(false);
  const [isInstalling, setIsInstalling] = useState(false);

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
      notify.error(getResendProfileErrorMessage(e), { response: e });
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

  const onInstallProfile = async () => {
    if (!onInstall) return;
    setIsInstalling(true);
    try {
      await onInstall(profile);
    } catch (e) {
      notify.error("Couldn't install. Please try again.", { response: e });
    }
    setIsInstalling(false);
  };

  const isFailed = profile.status === "failed";
  const isVerified = profile.status === "verified";
  // Unlike Windows/Apple profiles, an Android cert can get stuck mid-delivery
  // (e.g. a silent SCEP failure) with no automatic path back to failed. The
  // backend allows resending from any status but pending, so let admins
  // retry from here too instead of being stuck until the next status change.
  const isAndroidCertStuckEnforcing =
    isAndroidCertificate &&
    (profile.status === "delivering" || profile.status === "delivered");
  const isVerifying = profile.status === "verifying";
  const isRecoveryLockRow =
    profile.profile_uuid === REC_LOCK_SYNTHETIC_PROFILE_UUID;
  const isHostNameRow =
    profile.profile_uuid === HOST_NAME_SYNTHETIC_PROFILE_UUID;

  // The host name row is a synthetic row resent through its own endpoint, so it
  // must not go through the profile-resend path above.
  const showResendButton =
    canResendProfiles &&
    (isFailed ||
      isVerified ||
      isAndroidCertStuckEnforcing ||
      (isVerifying && canResendWhileVerifying)) &&
    !isRecoveryLockRow &&
    !isHostNameRow;
  // Disabled rather than hidden so users can tell resend exists for this profile.
  const isSelfService = canManageSelfServiceProfiles && profile.self_service;
  const isNotInstalled = profile.status === null;
  const isPendingInstall =
    profile.operation_type === "install" && profile.status === "pending";
  // Once Install is clicked it flips to a disabled Resend until the profile lands.
  const showInstallButton =
    isSelfService && !!onInstall && isNotInstalled && !isActionRequested;
  const showDisabledResendButton =
    canResendProfiles &&
    !showResendButton &&
    (profile.status !== null || (isSelfService && isActionRequested)) &&
    !isRecoveryLockRow &&
    !isHostNameRow;
  const showUninstallButton =
    isSelfService && !!onClickUninstall && !isNotInstalled && !isPendingInstall;
  const showRotateButton =
    canRotateRecoveryLockPassword && (isFailed || isVerified);
  // canResendHostNameTemplate is already pre-gated on the host name row by the
  // caller, mirroring how showRotateButton relies on canRotateRecoveryLockPassword.
  const showResendHostNameButton =
    canResendHostNameTemplate && (isFailed || isVerified);

  const renderPrimaryAction = () => {
    if (showInstallButton) {
      return (
        <ActionButton
          size={buttonSize}
          className={`${baseClass}__install-button`}
          text="Install"
          pendingText="Installing..."
          icon="install-self-service"
          isPending={isInstalling}
          disabled={isActionRequested}
          onClick={onInstallProfile}
        />
      );
    }
    if (showResendButton || showResendHostNameButton) {
      return (
        <ActionButton
          size={buttonSize}
          className={classnames(`${baseClass}__resend-button`, {
            [`${baseClass}__resending`]: isResending,
          })}
          text="Resend"
          pendingText="Resending..."
          icon="refresh"
          isPending={isResending}
          onClick={
            showResendButton ? onResendProfile : onResendHostNameTemplate
          }
        />
      );
    }
    if (showRotateButton) {
      return (
        <ActionButton
          size={buttonSize}
          className={classnames(`${baseClass}__rotate-button`, {
            [`${baseClass}__rotating`]: isRotating,
          })}
          text="Rotate"
          pendingText="Rotating..."
          icon="refresh"
          isPending={isRotating}
          onClick={onRotatePassword}
        />
      );
    }
    if (lacksResendPermission && profile.status !== null) {
      return (
        <ActionButton
          size={buttonSize}
          className={`${baseClass}__resend-button`}
          text="Resend"
          icon="refresh"
          disabled
          tooltip={NO_RESEND_PERMISSION_TOOLTIP_MESSAGE}
        />
      );
    }
    if (showDisabledResendForAndroidProfile || showDisabledResendButton) {
      return (
        <ActionButton
          size={buttonSize}
          className={`${baseClass}__resend-button`}
          text="Resend"
          icon="refresh"
          disabled
          tooltip={
            showDisabledResendForAndroidProfile
              ? ANDROID_PROFILE_NO_RESEND_TOOLTIP_MESSAGE
              : PROFILE_NO_RESEND_TOOLTIP_MESSAGE
          }
        />
      );
    }
    return null;
  };

  return (
    <div className={baseClass}>
      {renderPrimaryAction()}
      {showUninstallButton && (
        <ActionButton
          size={buttonSize}
          className={`${baseClass}__uninstall-button`}
          text="Uninstall"
          icon="trash"
          // A removal is already under way.
          disabled={isActionRequested || profile.operation_type === "remove"}
          onClick={() => onClickUninstall?.(profile)}
        />
      )}
    </div>
  );
};

export default OSSettingsActionsCell;
