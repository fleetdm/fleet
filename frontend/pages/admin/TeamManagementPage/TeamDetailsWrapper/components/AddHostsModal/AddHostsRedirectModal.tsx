import React from "react";
import Button from "components/buttons/Button";
import Modal from "components/modals/Modal";

interface IAddHostsModalProps {
  onCancel: () => void;
  onSubmit: () => void;
}

const baseClass = "add-hosts-redirect-modal";

const AddHostsRedirectModal = (props: IAddHostsModalProps): JSX.Element => {
  const { onCancel, onSubmit } = props;

  return (
    <Modal title={"Add hosts"} onExit={onCancel} className={baseClass}>
      <div className={`${baseClass}__modal-body`}>
        <p>
          Head to the Hosts page to transfer hosts that are already enrolled to
          Fleet or enroll new hosts.
        </p>
        <div className={`${baseClass}__btn-wrap`}>
          <Button
            className={`${baseClass}__btn`}
            type="button"
            variant="brand"
            onClick={onSubmit}
          >
            Go to Hosts page
          </Button>
          <Button
            className={`${baseClass}__btn`}
            onClick={onCancel}
            variant="inverse"
          >
            Cancel
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default AddHostsRedirectModal;
