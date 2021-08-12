import React, { useState } from "react";
import { size } from "lodash";

import { IQueryFormData, IQueryFormFields } from "interfaces/query";

// @ts-ignore
import Form from "components/forms/Form";
import Checkbox from "components/forms/fields/Checkbox"; // @ts-ignore
import InputField from "components/forms/fields/InputField"; // @ts-ignore
import Button from "components/buttons/Button";
import Modal from "components/modals/Modal";

interface INewQueryModalProps {
  baseClass: string;
  fields: IQueryFormFields;
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

  const handleUpdate = (evt: React.MouseEvent<HTMLButtonElement>) => {
    evt.preventDefault();

    const { description, name, observer_can_run } = fields;
    const { valid, errors: newErrors } = validateQueryName(
      name.value as string
    );
    setErrors({
      ...errors,
      ...newErrors,
    });

    valid &&
      onCreateQuery({
        description: description.value,
        name: name.value,
        query: queryValue,
        observer_can_run: observer_can_run.value,
      });
  };

  return (
    <Modal title={"Save query"} onExit={() => setIsSaveModalOpen(false)}>
      <form className={`${baseClass}__save-modal-form`}>
        <InputField
          {...fields.name}
          error={fields.name.error || errors.name}
          inputClassName={`${baseClass}__query-name`}
          label="Name"
          placeholder="What is your query called?"
        />
        <InputField
          {...fields.description}
          inputClassName={`${baseClass}__query-description`}
          label="Description"
          type="textarea"
          placeholder="What information does your query reveal?"
        />
        <Checkbox
          {...fields.observer_can_run}
          value={!!fields.observer_can_run.value}
          wrapperClassName={`${baseClass}__query-observer-can-run-wrapper`}
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
  fields: ["description", "name", "query", "observer_can_run"],
  validate: validateQueryName,
});
