import React from "react";

import configProfilesAPI from "services/entities/config_profiles";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import { notify } from "components/ToastNotification";

const baseClass = "resend-config-profile-modal";

interface IResendConfigProfileModalProps {
  name: string;
  uuid: string;
  count: number;
  onExit: () => void;
}

const ResendConfigProfileModal = ({
  name,
  uuid,
  count,
  onExit,
}: IResendConfigProfileModalProps) => {
  const [isResending, setIsResending] = React.useState(false);

  const countText = `${count} ${count === 1 ? "host" : "hosts"}`;

  const onClickResend = async () => {
    setIsResending(true);
    try {
      await configProfilesAPI.batchResendConfigProfile(uuid);
      notify.success(
        <>
          Resent the <b>{name}</b> configuration profile.
        </>
      );
      onExit();
    } catch (error) {
      notify.error(
        "Couldn't resend the configuration profile. Please try again.",
        { response: error }
      );
    }
    setIsResending(false);
  };

  return (
    <Modal
      className={baseClass}
      title="Resend configuration profile"
      onExit={onExit}
    >
      <p>
        This action will resend the <b>{name}</b> configuration profile to{" "}
        <b>{countText}</b>. To cancel after resending, delete and re-add the
        profile.
      </p>
      <div className="modal-cta-wrap">
        <Button
          onClick={onClickResend}
          isLoading={isResending}
          disabled={isResending}
        >
          Resend
        </Button>
        <Button variant="secondary" onClick={onExit} disabled={isResending}>
          Cancel
        </Button>
      </div>
    </Modal>
  );
};

export default ResendConfigProfileModal;
