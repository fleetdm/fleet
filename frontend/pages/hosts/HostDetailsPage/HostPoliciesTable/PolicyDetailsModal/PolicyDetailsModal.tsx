import React from "react";
import Button from "components/buttons/Button";
import Modal from "components/modals/Modal";

interface IPolicyDetailsProps {
  onCancel: () => void;
}

const baseClass = "policy-details-modal";

const PolicyDetailsModal = (props: IPolicyDetailsProps): JSX.Element => {
  const { onCancel } = props;

  return (
    <Modal title={"Policy Name"} onExit={onCancel} className={baseClass}>
      <div className={`${baseClass}__modal-body`}>
        <p>
          Lorem ipsum dolor sit amet, consectetur adipiscing elit. Maecenas
          feugiat venenatis quam, nec eleifend nisi aliquet non. Sed feugiat
          rutrum turpis, ac convallis odio egestas sit amet. Fusce vel sem
          massa. Quisque porttitor metus id vulputate vehicula. Donec ut nunc
          tempor, pretium lorem et, tempus est.
        </p>
        <div className={`${baseClass}__btn-wrap`}>
          <Button
            className={`${baseClass}__btn`}
            onClick={onCancel}
            variant="brand"
          >
            Cancel
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default PolicyDetailsModal;
