import React from "react";
import { useMutation } from "react-query";

import { ICustomHostVital } from "interfaces/custom_host_vitals";
import { hasStatusKey } from "interfaces/errors";
import customHostVitalsAPI from "services/entities/custom_host_vitals";
import { notify } from "components/ToastNotification";
import Modal from "components/Modal";
import Button from "components/buttons/Button";

interface IDeleteCustomHostVitalModalProps {
  vital: ICustomHostVital;
  onExit: () => void;
  onDelete: () => void;
}

const baseClass = "delete-custom-host-vital-modal";

const DeleteCustomHostVitalModal = ({
  vital,
  onExit,
  onDelete,
}: IDeleteCustomHostVitalModalProps) => {
  const { mutate: deleteCustomHostVital, isLoading: isDeleting } = useMutation(
    () => customHostVitalsAPI.deleteCustomHostVital(vital.id),
    {
      onSuccess: () => {
        notify.success("Custom host vital successfully deleted.");
        onDelete();
      },
      onError: (error) => {
        // TODO(#48559): once the backend reports the specific reference on a
        // 409, surface its detail instead of this generic copy, e.g.
        // "Couldn't delete. Host vital is referenced in label criteria for the
        // Envoy iPads label."
        const message =
          hasStatusKey(error) && error.status === 409
            ? "This custom host vital is referenced in a configuration profile or script and can't be deleted. To resolve, edit the configuration profile or script."
            : "An error occurred while deleting the custom host vital. Please try again.";
        notify.error(message, { response: error });
        onExit();
      },
    }
  );

  const onClickDelete = () => {
    deleteCustomHostVital();
  };

  return (
    <Modal
      title="Delete custom host vital"
      onExit={onExit}
      className={baseClass}
    >
      <div className={`${baseClass}__message`}>
        <span>
          Are you sure you want to delete the <b>{vital.name}</b> host vital?
        </span>
        <br />
        <br />
        Any references to the <b>{`$FLEET_HOST_VITAL_${vital.id}`}</b> variable
        will break.
      </div>
      <div className="modal-cta-wrap">
        <Button
          variant="alert"
          onClick={onClickDelete}
          isLoading={isDeleting}
          disabled={isDeleting}
        >
          Delete
        </Button>
        <Button variant="secondary" onClick={onExit}>
          Cancel
        </Button>
      </div>
    </Modal>
  );
};

export default DeleteCustomHostVitalModal;
