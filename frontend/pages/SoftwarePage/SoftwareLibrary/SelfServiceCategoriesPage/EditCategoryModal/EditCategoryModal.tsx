import React, { useState } from "react";

import selfServiceCategoriesAPI from "services/entities/self_service_categories";
import { hasStatusKey } from "interfaces/errors";
import { ISelfServiceCategory } from "interfaces/self_service_category";
import { MAX_ENTITY_NAME_LENGTH } from "utilities/constants";

import Button from "components/buttons/Button";
import InputField from "components/forms/fields/InputField";
import Modal from "components/Modal";

const baseClass = "edit-category-modal";

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
    trimmedName.length === 0 || trimmedName.length > MAX_ENTITY_NAME_LENGTH;
  const isUnchanged = trimmedName === category.name;
  const isDisabled = isInvalid || isUnchanged || isSubmitting;

  const onNameChange = (value: string) => {
    setName(value);
    if (error) setError(null);
  };

  const onSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    if (isDisabled) return;

    setIsSubmitting(true);
    try {
      await selfServiceCategoriesAPI.updateCategory(category.id, {
        name: trimmedName,
      });
      onSuccess();
    } catch (e) {
      if (hasStatusKey(e) && e.status === 409) {
        setError(
          "A self-service category with this name already exists in this fleet."
        );
      } else {
        setError("Couldn't update self-service category.");
      }
      setIsSubmitting(false);
    }
  };

  return (
    <Modal
      title="Edit category"
      onExit={onExit}
      className={baseClass}
      isContentDisabled={isSubmitting}
    >
      <form className={`${baseClass}__form`} onSubmit={onSubmit}>
        <InputField
          label="Name"
          name="name"
          value={name}
          onChange={onNameChange}
          error={error}
          autofocus
          ignore1password
          inputOptions={{ maxLength: MAX_ENTITY_NAME_LENGTH }}
        />
        <div className="modal-cta-wrap">
          <Button type="submit" disabled={isDisabled} isLoading={isSubmitting}>
            Save
          </Button>
          <Button variant="secondary" onClick={onExit} disabled={isSubmitting}>
            Cancel
          </Button>
        </div>
      </form>
    </Modal>
  );
};

export default EditCategoryModal;
