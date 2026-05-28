import React, { useState } from "react";

import selfServiceCategoriesAPI from "services/entities/self_service_categories";
import { hasStatusKey } from "interfaces/errors";

import Button from "components/buttons/Button";
import InputField from "components/forms/fields/InputField";
import Modal from "components/Modal";

const baseClass = "add-category-modal";
const NAME_MAX_LENGTH = 255;

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
      await selfServiceCategoriesAPI.add({
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
        setError("Couldn’t add self-service category.");
      }
      setIsSubmitting(false);
    }
  };

  return (
    <Modal title="Add category" onExit={onExit} className={baseClass}>
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
            Add
          </Button>
          <Button variant="inverse" onClick={onExit}>
            Cancel
          </Button>
        </div>
      </form>
    </Modal>
  );
};

export default AddCategoryModal;
