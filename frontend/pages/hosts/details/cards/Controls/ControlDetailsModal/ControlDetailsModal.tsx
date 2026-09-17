import React from "react";

import Button from "components/buttons/Button";
import CopyButton from "components/buttons/CopyButton";
import Icon from "components/Icon";
import Modal from "components/Modal";
import Textarea from "components/Textarea";

import OSSettingsResendCell from "../OSSettingsResendCell";
import { getDetailGuidance, getDetailText } from "../detailFormatting";
import {
  getRowActionProps,
  IHostMdmProfileWithAddedStatus,
} from "../OSSettingsTableConfig";
import {
  getControlDisplayOption,
  isDiskEncryptionProfile,
} from "../statusDisplayConfig";

const baseClass = "control-details-modal";

interface IControlDetailsModalProps {
  control: IHostMdmProfileWithAddedStatus;
  hostDisplayName: string;
  /** True on the My device page, which addresses the end user directly. */
  isDeviceUser?: boolean;
  /** Fleet setting for macOS: disk encryption enforced without key escrow. */
  isMacOSDiskEncryptionEnforceOnly?: boolean;
  canResendProfiles: boolean;
  canRotateRecoveryLockPassword?: boolean;
  canResendHostNameTemplate?: boolean;
  resendRequest: (profileUUID: string) => Promise<void>;
  resendCertificateRequest?: (certificateTemplateId: number) => Promise<void>;
  rotateRecoveryLockPassword?: () => Promise<void>;
  resendHostNameTemplate?: () => Promise<void>;
  /** Fires after a successful resend/rotate, which also closes the modal. */
  onProfileResent: () => void;
  onExit: () => void;
}

const ControlDetailsModal = ({
  control,
  hostDisplayName,
  isDeviceUser = false,
  isMacOSDiskEncryptionEnforceOnly = false,
  canResendProfiles,
  canRotateRecoveryLockPassword,
  canResendHostNameTemplate,
  resendRequest,
  resendCertificateRequest,
  rotateRecoveryLockPassword,
  resendHostNameTemplate,
  onProfileResent,
  onExit,
}: IControlDetailsModalProps) => {
  const displayOption = getControlDisplayOption(control);
  const detailText = getDetailText(control);
  const guidance = getDetailGuidance(control);

  // A verified control's detail contradicts the status copy rather than adding
  // to it (an activation predicate excluded the host), so it replaces the
  // sentence. Other statuses keep theirs — the detail may describe a past
  // attempt, since resending only nulls the status until the next cron run.
  //
  // Disk encryption on "Action required" is the other exception. The generic
  // copy for that status names a BitLocker PIN, but the server reaches the same
  // status for several unrelated reasons (the agent could not add a TPM
  // protector, the TPM is not ready, a restart is staged) and sends the actual
  // reason in the detail. Preferring the detail keeps the reason in one place,
  // server-side, instead of asserting a PIN that may not be required at all.
  const detailReplacesMessage =
    !!detailText &&
    (displayOption?.statusText === "Verified" ||
      (displayOption?.statusText === "Action required" &&
        isDiskEncryptionProfile(control.name)));

  const renderMessage = () => {
    if (detailReplacesMessage) {
      return detailText;
    }

    const { message } = displayOption ?? {};
    if (!message) {
      return null;
    }

    return typeof message === "string"
      ? message
      : message({
          hostDisplayName,
          settingName: control.name,
          isDiskEncryptionProfile: isDiskEncryptionProfile(control.name),
          isDeviceUser,
          isMacOSDiskEncryptionEnforceOnly,
          profileUUID: control.profile_uuid,
        });
  };

  const message = renderMessage();

  const rowActions = getRowActionProps(
    control,
    canResendProfiles,
    canRotateRecoveryLockPassword,
    canResendHostNameTemplate
  );

  // Guidance that quotes the detail makes the block below it redundant.
  const showDetailBlock =
    !!detailText && !detailReplacesMessage && !guidance?.supersedesDetail;

  return (
    <Modal title="Details" onExit={onExit} className={baseClass}>
      <>
        <div className={`${baseClass}__modal-content`}>
          {message && (
            <div className={`${baseClass}__status`}>
              {displayOption && <Icon name={displayOption.iconName} />}
              <span className={`${baseClass}__status-message`}>{message}</span>
            </div>
          )}
          {guidance && (
            <div className={`${baseClass}__guidance`}>{guidance.message}</div>
          )}
          {showDetailBlock && (
            <Textarea
              variant="code"
              className={`${baseClass}__details`}
              label={
                <div className={`${baseClass}__details-label`}>
                  <span>Details:</span>
                  <CopyButton
                    copyText={detailText}
                    variant="compact"
                    ariaLabel="Copy details"
                  />
                </div>
              }
            >
              {detailText}
            </Textarea>
          )}
        </div>
        {/* .modal-cta-wrap is row-reverse, so the primary action comes first. */}
        <div className="modal-cta-wrap">
          <Button onClick={onExit}>Close</Button>
          <OSSettingsResendCell
            canResendProfiles={rowActions.canResendProfiles}
            canRotateRecoveryLockPassword={
              rowActions.canRotateRecoveryLockPassword
            }
            canResendHostNameTemplate={rowActions.canResendHostNameTemplate}
            profile={control}
            resendRequest={resendRequest}
            resendCertificateRequest={resendCertificateRequest}
            rotateRecoveryLockPassword={rotateRecoveryLockPassword}
            resendHostNameTemplate={resendHostNameTemplate}
            onProfileResent={onProfileResent}
          />
        </div>
      </>
    </Modal>
  );
};

export default ControlDetailsModal;
