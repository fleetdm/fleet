import React, { useState } from "react";

import hostAPI from "services/entities/hosts";

import { notify } from "components/ToastNotification";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
import InputField from "components/forms/fields/InputField";

const baseClass = "unlock-user-account-modal";

interface IUnlockUserAccountModalProps {
  id: number;
  hostName: string;
  onExit: () => void;
  onSuccess?: () => void;
}

const UnlockUserAccountModal = ({
  id,
  hostName,
  onExit,
  onSuccess,
}: IUnlockUserAccountModalProps) => {
  const [username, setUsername] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);

  const trimmedUsername = username.trim();

  const onUnlock = async () => {
    if (!trimmedUsername || isSubmitting) return;

    setIsSubmitting(true);
    try {
      await hostAPI.unlockUserAccount(id, trimmedUsername);
      notify.success(
        `Successfully sent request to unlock ${trimmedUsername} on ${hostName}.`
      );
      onSuccess?.();
      onExit();
    } catch (e) {
      notify.error(
        "Couldn't send request to unlock this user account. Please try again.",
        { response: e }
      );
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Modal title="Unlock user account" onExit={onExit} className={baseClass}>
      <p>
        Unlock a local macOS user account after too many failed login attempts.
        This doesn&apos;t reset the password or unlock FileVault.
      </p>
      <InputField
        autofocus
        label="Username"
        name="username"
        value={username}
        onChange={setUsername}
        placeholder="Local account username"
        inputOptions={{ required: true }}
      />
      <div className="modal-cta-wrap">
        <Button
          type="button"
          onClick={onUnlock}
          isLoading={isSubmitting}
          disabled={!trimmedUsername || isSubmitting}
        >
          Unlock user
        </Button>
        <Button onClick={onExit} variant="secondary" disabled={isSubmitting}>
          Cancel
        </Button>
      </div>
    </Modal>
  );
};

export default UnlockUserAccountModal;
