import React, { useState } from "react";
import { size } from "lodash";

import { IQueryFormData, IQueryFormFieldsModal } from "interfaces/query";

// @ts-ignore
import Form from "components/forms/Form";
import Checkbox from "components/forms/fields/Checkbox"; // @ts-ignore
import InputField from "components/forms/fields/InputField"; // @ts-ignore
import Button from "components/buttons/Button";
import Modal from "components/modals/Modal";
import { useDeepEffect } from "utilities/hooks";

export interface INewQueryModalProps {
  baseClass: string;
  fields: IQueryFormFieldsModal;
  queryValue: string;
  onCreateQuery: (formData: IQueryFormData) => void;
  setIsSaveModalOpen: (isOpen: boolean) => void;
}

const validateQueryName = (name: string) => {
  const errors: { [key: string]: any } = {};

  if (!name) {
    errors.name = "Query name must be present";
  }

  const valid = !size(errors);
  return { valid, errors };
};

const NewQueryModal = ({
  baseClass,
  fields,
  queryValue,
  onCreateQuery,
  setIsSaveModalOpen,
}: INewQueryModalProps) => {
  const [errors, setErrors] = useState<{ [key: string]: any }>({});

  useDeepEffect(() => {
    if (fields.nameModal.value) {
      setErrors({});
    }
  }, [fields]);

  const handleUpdate = (evt: React.MouseEvent<HTMLButtonElement>) => {
    evt.preventDefault();

    const { descriptionModal, nameModal, observer_can_run_modal } = fields;
    const { valid, errors: newErrors } = validateQueryName(
      nameModal.value as string
    );
    setErrors({
      ...errors,
      ...newErrors,
    });

    valid &&
      onCreateQuery({
        description: descriptionModal.value,
        name: nameModal.value,
        query: queryValue,
        observer_can_run: observer_can_run_modal.value,
      });

    setIsSaveModalOpen(false);
  };

  return (
    <Modal title={"Save query"} onExit={() => setIsSaveModalOpen(false)}>
      <form className={`${baseClass}__save-modal-form`}>
        <InputField
          {...fields.nameModal}
          error={fields.nameModal.error || errors.name}
          inputClassName={`${baseClass}__query-save-modal-name`}
          label="Name"
          placeholder="What is your query called?"
        />
        <InputField
          {...fields.descriptionModal}
          inputClassName={`${baseClass}__query-save-modal-description`}
          label="Description"
          type="textarea"
          placeholder="What information does your query reveal?"
        />
        <Checkbox
          {...fields.observer_can_run_modal}
          value={!!fields.observer_can_run_modal.value}
          wrapperClassName={`${baseClass}__query-save-modal-observer-can-run-wrapper`}
        >
          Observers can run
        </Checkbox>
        <p>
          Users with the Observer role will be able to run this query on hosts
          where they have access.
        </p>
        <hr />
        <div
          className={`${baseClass}__button-wrap ${baseClass}__button-wrap--modal`}
        >
          <Button
            className={`${baseClass}__btn`}
            onClick={() => setIsSaveModalOpen(false)}
            variant="text-link"
          >
            Cancel
          </Button>
          <Button
            className={`${baseClass}__btn`}
            type="button"
            variant="brand"
            onClick={handleUpdate}
          >
            Save query
          </Button>
        </div>
      </form>
    </Modal>
  );
};

export default Form(NewQueryModal, {
  fields: [
    "descriptionModal",
    "nameModal",
    "queryModal",
    "observer_can_run_modal",
  ],
  validate: validateQueryName,
});
