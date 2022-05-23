import React, { useState, useCallback, useEffect } from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import InfoBanner from "components/InfoBanner/InfoBanner";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Spinner from "components/Spinner";

const baseClass = "create-team-modal";

export interface ICreateTeamFormData {
  name: string;
}

interface ICreateTeamModalProps {
  onCancel: () => void;
  onSubmit: (formData: ICreateTeamFormData) => void;
  backendValidators: { [key: string]: string };
  isLoading: boolean;
}

const CreateTeamModal = ({
  onCancel,
  onSubmit,
  backendValidators,
  isLoading,
}: ICreateTeamModalProps): JSX.Element => {
  const [name, setName] = useState("");
  const [errors, setErrors] = useState<{ [key: string]: string }>(
    backendValidators
  );

  useEffect(() => {
    setErrors(backendValidators);
  }, [backendValidators]);

  const onInputChange = useCallback(
    (value: string) => {
      setName(value);
      setErrors({});
    },
    [setName]
  );

  const onFormSubmit = useCallback(
    (evt) => {
      evt.preventDefault();
      onSubmit({
        name,
      });
    },
    [onSubmit, name]
  );

  return (
    <Modal title={"Create team"} onExit={onCancel} className={baseClass}>
      {isLoading ? (
        <Spinner />
      ) : (
        <form
          className={`${baseClass}__form`}
          onSubmit={onFormSubmit}
          autoComplete="off"
        >
          <InputField
            autofocus
            name="name"
            onChange={onInputChange}
            label="Team name"
            placeholder="Team name"
            value={name}
            error={errors.name}
          />
          <InfoBanner className={`${baseClass}__sandbox-info`}>
            <p className={`${baseClass}__info-header`}>
              Need to test queries and configurations before deploying?
            </p>
            <p>
              A popular pattern is to end a team’s name with “- Sandbox”, then
              you can use this to test new queries and configuration with
              staging hosts or volunteers acting as canaries.
            </p>
          </InfoBanner>
          <div className="modal-cta-wrap">
            <Button onClick={onCancel} variant="inverse">
              Cancel
            </Button>
            <Button type="submit" variant="brand" disabled={name === ""}>
              Create
            </Button>
          </div>
        </form>
      )}
    </Modal>
  );
};

export default CreateTeamModal;
