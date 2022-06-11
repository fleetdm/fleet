import React, { useEffect } from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import Spinner from "components/Spinner";

const baseClass = "remove-policies-modal";

interface IRemovePoliciesModalProps {
  isLoading: boolean;
  onCancel: () => void;
  onSubmit: () => void;
}

const RemovePoliciesModal = ({
  isLoading,
  onCancel,
  onSubmit,
}: IRemovePoliciesModalProps): JSX.Element => {
  useEffect(() => {
    const listener = (event: KeyboardEvent) => {
      if (event.code === "Enter" || event.code === "NumpadEnter") {
        event.preventDefault();
        onSubmit();
      }
    };
    document.addEventListener("keydown", listener);
    return () => {
      document.removeEventListener("keydown", listener);
    };
  }, []);

  return (
    <Modal title={"Delete policies"} onExit={onCancel} className={baseClass}>
      <>
        {isLoading ? (
          <Spinner />
        ) : (
          <div className={baseClass}>
            Are you sure you want to delete the selected policies?
            <div className={`${baseClass}__btn-wrap`}>
              <Button
                className={`${baseClass}__btn`}
                type="button"
                variant="alert"
                onClick={onSubmit}
              >
                Delete
              </Button>
              <Button
                className={`${baseClass}__btn`}
                onClick={onCancel}
                variant="inverse-alert"
              >
                Cancel
              </Button>
            </div>
          </div>
        )}
      </>
    </Modal>
  );
};

export default RemovePoliciesModal;
