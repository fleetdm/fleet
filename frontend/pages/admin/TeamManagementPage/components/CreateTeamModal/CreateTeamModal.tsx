import React, { useState, useCallback } from "react";

import Modal from "components/modals/Modal";
import Button from "components/buttons/Button";
import InfoBanner from "components/InfoBanner/InfoBanner";
// @ts-ignore
import InputFieldWithIcon from "components/forms/fields/InputFieldWithIcon";

const baseClass = "create-team-modal";

export interface ICreateTeamFormData {
  name: string;
}

interface ICreateTeamModalProps {
  onCancel: () => void;
  onSubmit: (formData: ICreateTeamFormData) => void;
}

const CreateTeamModal = (props: ICreateTeamModalProps): JSX.Element => {
  const { onCancel, onSubmit } = props;

  const [name, setName] = useState("");

  const onInputChange = useCallback(
    (value: string) => {
      setName(value);
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
      <form className={`${baseClass}__form`}>
        <InputFieldWithIcon
          autofocus
          // error={errors.name}
          name="name"
          onChange={onInputChange}
          placeholder="Team name"
          value={name}
        />
        <InfoBanner className={`${baseClass}__sandbox-info`}>
          <p className={`${baseClass}__info-header`}>
            Need to test queries and configurations before deploying?
          </p>
          <p>
            A popular pattern is to end a team’s name with “- Sandbox”, then you
            can use this to test new queries and configuration with staging
            hosts or volunteers acting as canaries.
          </p>
        </InfoBanner>
        <div className={`${baseClass}__btn-wrap`}>
          <Button
            className={`${baseClass}__btn`}
            type="button"
            variant="brand"
            onClick={onFormSubmit}
          >
            Create
          </Button>
          <Button
            className={`${baseClass}__btn`}
            onClick={onCancel}
            variant="inverse"
          >
            Cancel
          </Button>
        </div>
      </form>
    </Modal>
  );
};

export default CreateTeamModal;
