import React, { useState } from "react";

import selfServiceCategoriesAPI from "services/entities/self_service_categories";
import { ISelfServiceCategory } from "interfaces/self_service_category";

import { notify } from "components/ToastNotification";
import Button from "components/buttons/Button";
import Modal from "components/Modal";

const baseClass = "delete-category-modal";

interface IDeleteCategoryModalProps {
  category: ISelfServiceCategory;
  onExit: () => void;
  onSuccess: () => void;
}

const DeleteCategoryModal = ({
  category,
  onExit,
  onSuccess,
}: IDeleteCategoryModalProps) => {
  const [isDeleting, setIsDeleting] = useState(false);

  const onDelete = async () => {
    if (isDeleting) return;
    setIsDeleting(true);
    try {
      await selfServiceCategoriesAPI.deleteCategory(category.id);
      onSuccess();
    } catch (e) {
      notify.error("Couldn't delete self-service category.", { response: e });
      setIsDeleting(false);
    }
  };

  return (
    <Modal
      title="Delete category"
      onExit={onExit}
      className={baseClass}
      isContentDisabled={isDeleting}
    >
      <>
        <p className={`${baseClass}__body`}>
          The category will be removed from all associated software.
        </p>
        <div className="modal-cta-wrap">
          <Button
            variant="alert"
            disabled={isDeleting}
            isLoading={isDeleting}
            onClick={onDelete}
          >
            Delete
          </Button>
          <Button variant="secondary" onClick={onExit} disabled={isDeleting}>
            Cancel
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default DeleteCategoryModal;
