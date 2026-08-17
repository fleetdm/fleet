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
  const isFailed = control.status === "failed";

  const renderMessage = () => {
    // A backend-provided detail on a control that hasn't failed is more specific
    // than the generic status copy — an Android profile waiting on its
    // certificate, or a custom activation that deliberately excluded this host.
    // Show it in place of the status sentence rather than alongside it.
    if (!isFailed && detailText) {
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
        });
  };

  const message = renderMessage();

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
            <div className={`${baseClass}__guidance`}>{guidance}</div>
          )}
          {isFailed && detailText && (
            <Textarea
              variant="code"
              className={`${baseClass}__details`}
              label={
                <div className={`${baseClass}__details-label`}>
                  <span>Details:</span>
                  <CopyButton
                    copyText={detailText}
                    size="small"
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
            {...getRowActionProps(
              control,
              canResendProfiles,
              canRotateRecoveryLockPassword,
              canResendHostNameTemplate
            )}
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
