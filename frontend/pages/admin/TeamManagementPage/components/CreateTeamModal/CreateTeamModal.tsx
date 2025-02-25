import React, { useState, useCallback, useEffect } from "react";

import { ITeamFormData } from "services/entities/teams";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import InfoBanner from "components/InfoBanner/InfoBanner";
// @ts-ignore
import InputField from "components/forms/fields/InputField";

const baseClass = "create-team-modal";

interface ICreateTeamModalProps {
  onCancel: () => void;
  onSubmit: (formData: ITeamFormData) => void;
  backendValidators: { [key: string]: string };
  isUpdatingTeams: boolean;
}

const CreateTeamModal = ({
  onCancel,
  onSubmit,
  backendValidators,
  isUpdatingTeams,
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
    (evt: any) => {
      evt.preventDefault();
      onSubmit({
        name: name.trim(),
      });
    },
    [onSubmit, name]
  );

  return (
    <Modal title="Create team" onExit={onCancel} className={baseClass}>
      <form
        className={`${baseClass}__form`}
        onSubmit={onFormSubmit}
        autoComplete="off"
      >
        <InputField
          autofocus
          name="name"
          onChange={onInputChange}
          onBlur={() => {
            setName(name.trim());
          }}
          label="Team name"
          placeholder="Workstations"
          value={name}
          error={errors.name}
          ignore1password
        />
        <InfoBanner className={`${baseClass}__sandbox-info`}>
          To organize your hosts, create a team, like
          &ldquo;Workstations,&rdquo; &ldquo;Servers,&rdquo; or &ldquo;Servers
          (canary)&rdquo;.
        </InfoBanner>
        <div className="modal-cta-wrap">
          <Button
            type="submit"
            variant="brand"
            disabled={name === ""}
            className="create-loading"
            isLoading={isUpdatingTeams}
          >
            Create
          </Button>
          <Button onClick={onCancel} variant="inverse">
            Cancel
          </Button>
        </div>
      </form>
    </Modal>
  );
};

export default CreateTeamModal;
