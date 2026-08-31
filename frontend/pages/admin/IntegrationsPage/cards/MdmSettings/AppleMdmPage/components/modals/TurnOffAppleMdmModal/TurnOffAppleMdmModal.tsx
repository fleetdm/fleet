import React, { useCallback, useState } from "react";

import Button from "components/buttons/Button";
import InputField from "components/forms/fields/InputField";
import Modal from "components/Modal";

const baseClass = "modal turn-off-apple-mdm-modal";
const bemClass = "turn-off-apple-mdm-modal";

interface ITurnOffAppleMdmModalProps {
  serverUrl: string;
  onCancel: () => void;
  onConfirm: () => void;
}

const TurnOffAppleMdmModal = ({
  serverUrl,
  onConfirm,
  onCancel,
}: ITurnOffAppleMdmModalProps): JSX.Element => {
  const [isDeleting, setIsDeleting] = useState(false);
  const [enteredUrl, setEnteredUrl] = useState("");

  const onClickConfirm = useCallback(() => {
    setIsDeleting(true);
    onConfirm();
  }, [onConfirm]);

  return (
    <Modal title="Turn off MDM" onExit={onCancel} className={baseClass}>
      <div className={baseClass}>
        <p>
          If you want to use MDM features again, you&apos;ll have to upload a
          new APNs certificate and all end users will have to turn MDM off and
          back on.
        </p>
        <p>
          To confirm, enter your Fleet URL: <b>{serverUrl}</b>
        </p>
        <InputField
          autofocus
          inputWrapperClass={`${bemClass}__url-input`}
          placeholder="https://fleet.example.com"
          value={enteredUrl}
          onChange={(val: string) => setEnteredUrl(val)}
        />
        <div className="modal-cta-wrap">
          <Button
            type="button"
            variant="alert"
            onClick={onClickConfirm}
            isLoading={isDeleting}
            disabled={isDeleting || enteredUrl !== serverUrl}
          >
            Turn off
          </Button>
          <Button onClick={onCancel} disabled={isDeleting} variant="secondary">
            Cancel
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default TurnOffAppleMdmModal;
