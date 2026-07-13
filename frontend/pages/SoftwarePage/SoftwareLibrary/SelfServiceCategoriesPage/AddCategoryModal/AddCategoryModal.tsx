import React, { useState } from "react";

import selfServiceCategoriesAPI from "services/entities/self_service_categories";
import { hasStatusKey } from "interfaces/errors";
import { MAX_ENTITY_NAME_LENGTH } from "utilities/constants";

import Button from "components/buttons/Button";
import InputField from "components/forms/fields/InputField";
import Modal from "components/Modal";

const baseClass = "add-category-modal";

interface IAddCategoryModalProps {
  fleetId: number;
  onExit: () => void;
  onSuccess: () => void;
}

const AddCategoryModal = ({
  fleetId,
  onExit,
  onSuccess,
}: IAddCategoryModalProps) => {
  const [name, setName] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const trimmedName = name.trim();
  const isDisabled = trimmedName.length === 0 || isSubmitting;

  const onNameChange = (value: string) => {
    setName(value);
    if (error) setError(null);
  };

  const onSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    if (isDisabled) return;

    setIsSubmitting(true);
    try {
      await selfServiceCategoriesAPI.addCategory({
        fleet_id: fleetId,
        name: trimmedName,
      });
      onSuccess();
    } catch (e) {
      if (hasStatusKey(e) && e.status === 409) {
        setError(
          "A self-service category with this name already exists in this fleet."
        );
      } else {
        setError("Couldn't add self-service category.");
      }
      setIsSubmitting(false);
    }
  };

  return (
    <Modal
      title="Add category"
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
            Add
          </Button>
          <Button variant="secondary" onClick={onExit} disabled={isSubmitting}>
            Cancel
          </Button>
        </div>
      </form>
    </Modal>
  );
};

export default AddCategoryModal;
