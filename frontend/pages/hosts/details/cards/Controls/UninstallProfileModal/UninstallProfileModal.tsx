import React, { useState } from "react";

import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
import Modal from "components/Modal";
import ModalFooter from "components/ModalFooter";
import { notify } from "components/ToastNotification";

const baseClass = "uninstall-profile-modal";

interface IUninstallProfileModalProps {
  profileName: string;
  profileUUID: string;
  hostDisplayName: string;
  isDeviceUser?: boolean;
  uninstallRequest: (profileUUID: string) => Promise<void>;
  onSuccess: (profileUUID: string) => void;
  onExit: () => void;
}

const UninstallProfileModal = ({
  profileName,
  profileUUID,
  hostDisplayName,
  isDeviceUser = false,
  uninstallRequest,
  onSuccess,
  onExit,
}: IUninstallProfileModalProps) => {
  const [isChecked, setIsChecked] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const onUninstall = async () => {
    setIsSubmitting(true);
    try {
      await uninstallRequest(profileUUID);
      onSuccess(profileUUID);
    } catch (e) {
      notify.error("Couldn't uninstall. Please try again.", { response: e });
    }
    setIsSubmitting(false);
    onExit();
  };

  return (
    <Modal
      title="Uninstall configuration profile"
      className={baseClass}
      onExit={onExit}
    >
      <>
        <p>
          This removes anything <b>{profileName}</b> added (mail accounts,
          certificates, files). Reinstalling the profile later won&apos;t
          restore what&apos;s removed.
        </p>
        <div className={`${baseClass}__confirm`}>
          <p>
            <b>Please check to confirm:</b>
          </p>
          <Checkbox
            value={isChecked}
            onChange={(value: boolean) => setIsChecked(value)}
            variant="danger"
          >
            I understand this action can&apos;t be undone for{" "}
            {isDeviceUser ? "my device" : <b>{hostDisplayName}</b>}.
          </Checkbox>
        </div>
        <ModalFooter
          primaryButtons={
            <>
              <Button variant="secondary" onClick={onExit}>
                Cancel
              </Button>
              <Button
                variant="alert"
                disabled={!isChecked || isSubmitting}
                isLoading={isSubmitting}
                onClick={onUninstall}
              >
                Uninstall
              </Button>
            </>
          }
        />
      </>
    </Modal>
  );
};

export default UninstallProfileModal;
