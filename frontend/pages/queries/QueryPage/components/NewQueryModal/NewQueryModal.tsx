import React, { useState, useEffect } from "react";
import { size } from "lodash";

import { IQueryFormData } from "interfaces/query";
import useDeepEffect from "hooks/useDeepEffect";

import Checkbox from "components/forms/fields/Checkbox";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Button from "components/buttons/Button";
import Modal from "components/Modal";
import Spinner from "components/Spinner";

export interface INewQueryModalProps {
  baseClass: string;
  queryValue: string;
  isLoading: boolean;
  onCreateQuery: (formData: IQueryFormData) => void;
  setIsSaveModalOpen: (isOpen: boolean) => void;
  backendValidators: { [key: string]: string };
}

const validateQueryName = (name: string) => {
  const errors: { [key: string]: string } = {};

  if (!name) {
    errors.name = "Query name must be present";
  }

  const valid = !size(errors);
  return { valid, errors };
};

const NewQueryModal = ({
  baseClass,
  queryValue,
  isLoading,
  onCreateQuery,
  setIsSaveModalOpen,
  backendValidators,
}: INewQueryModalProps): JSX.Element => {
  const [name, setName] = useState<string>("");
  const [description, setDescription] = useState<string>("");
  const [observerCanRun, setObserverCanRun] = useState<boolean>(false);
  const [errors, setErrors] = useState<{ [key: string]: string }>(
    backendValidators
  );

  useDeepEffect(() => {
    if (name) {
      setErrors({});
    }
  }, [name]);

  useEffect(() => {
    setErrors(backendValidators);
  }, [backendValidators]);

  const handleUpdate = (evt: React.MouseEvent<HTMLFormElement>) => {
    evt.preventDefault();

    const { valid, errors: newErrors } = validateQueryName(name);
    setErrors({
      ...errors,
      ...newErrors,
    });

    if (valid) {
      onCreateQuery({
        description,
        name,
        query: queryValue,
        observer_can_run: observerCanRun,
      });
    }
  };

  return (
    <Modal title={"Save query"} onExit={() => setIsSaveModalOpen(false)}>
      <>
        <form
          onSubmit={handleUpdate}
          className={`${baseClass}__save-modal-form`}
          autoComplete="off"
        >
          <InputField
            name="name"
            onChange={(value: string) => setName(value)}
            value={name}
            error={errors.name}
            inputClassName={`${baseClass}__query-save-modal-name`}
            label="Name"
            placeholder="What is your query called?"
          />
          <InputField
            name="description"
            onChange={(value: string) => setDescription(value)}
            value={description}
            inputClassName={`${baseClass}__query-save-modal-description`}
            label="Description"
            type="textarea"
            placeholder="What information does your query reveal? (optional)"
          />
          <Checkbox
            name="observerCanRun"
            onChange={setObserverCanRun}
            value={observerCanRun}
            wrapperClassName={`${baseClass}__query-save-modal-observer-can-run-wrapper`}
          >
            Observers can run
          </Checkbox>
          <p>
            Users with the Observer role will be able to run this query on hosts
            where they have access.
          </p>
          <hr />
          <div className="modal-cta-wrap">
            <Button
              type="submit"
              variant="brand"
              className="save-query-loading"
              isLoading={isLoading}
            >
              Save query
            </Button>
            <Button onClick={() => setIsSaveModalOpen(false)} variant="inverse">
              Cancel
            </Button>
          </div>
        </form>
      </>
    </Modal>
  );
};

export default NewQueryModal;
