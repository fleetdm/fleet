import React, { useState } from "react";

import selfServiceCategoriesAPI from "services/entities/self_service_categories";
import { hasStatusKey } from "interfaces/errors";
import { ISelfServiceCategory } from "interfaces/self_service_category";

import Button from "components/buttons/Button";
import InputField from "components/forms/fields/InputField";
import Modal from "components/Modal";

const baseClass = "edit-category-modal";
const NAME_MAX_LENGTH = 255;

interface IEditCategoryModalProps {
  category: ISelfServiceCategory;
  onExit: () => void;
  onSuccess: () => void;
}

const EditCategoryModal = ({
  category,
  onExit,
  onSuccess,
}: IEditCategoryModalProps) => {
  const [name, setName] = useState(category.name);
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const trimmedName = name.trim();
  const isInvalid =
    trimmedName.length === 0 || trimmedName.length > NAME_MAX_LENGTH;
  const isDisabled = isInvalid || isSubmitting;

  const onNameChange = (value: string) => {
    setName(value);
    if (error) setError(null);
  };

  const onSubmit = async () => {
    if (isDisabled) return;

    setIsSubmitting(true);
    try {
      await selfServiceCategoriesAPI.edit(category.id, { name: trimmedName });
      onSuccess();
    } catch (e) {
      if (hasStatusKey(e) && e.status === 409) {
        setError(
          "A self-service category with this name already exists in this fleet."
        );
      } else {
        setError("Couldn’t update self-service category.");
      }
      setIsSubmitting(false);
    }
  };

  return (
    <Modal title="Edit category" onExit={onExit} className={baseClass}>
      <form
        className={`${baseClass}__form`}
        onSubmit={(e) => {
          e.preventDefault();
          onSubmit();
        }}
      >
        <InputField
          label="Name"
          name="name"
          value={name}
          onChange={onNameChange}
          error={error}
          autofocus
          ignore1password
          inputOptions={{ maxLength: NAME_MAX_LENGTH }}
        />
        <div className="modal-cta-wrap">
          <Button
            type="submit"
            disabled={isDisabled}
            isLoading={isSubmitting}
            onClick={onSubmit}
          >
            Save
          </Button>
          <Button variant="inverse" onClick={onExit}>
            Cancel
          </Button>
        </div>
      </form>
    </Modal>
  );
};

export default EditCategoryModal;
