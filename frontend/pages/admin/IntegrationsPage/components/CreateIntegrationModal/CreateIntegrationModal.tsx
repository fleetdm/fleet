import React, { useState, useCallback, useEffect } from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import InfoBanner from "components/InfoBanner/InfoBanner";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
// @ts-ignore
import FleetIcon from "components/icons/FleetIcon";

const baseClass = "create-integration-modal";

export interface ICreateIntegrationFormData {
  name: string;
}

interface ICreateIntegrationModalProps {
  onCancel: () => void;
  onSubmit: (formData: ICreateIntegrationFormData) => void;
  backendValidators: { [key: string]: string };
}

const CreateIntegrationModal = ({
  onCancel,
  onSubmit,
  backendValidators,
}: ICreateIntegrationModalProps): JSX.Element => {
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

  // TODO: Move this form to IntegrationForm.tsx and have that component used for create and edit integration modal
  return (
    <Modal title={"Add integration"} onExit={onCancel} className={baseClass}>
      <>
        <InfoBanner className={`${baseClass}__sandbox-info`}>
          <p className={`${baseClass}__info-header`}>
            Fleet supports Jira as a ticket destination.&nbsp;
            <a
              href="https://github.com/fleetdm/fleet/issues/new?assignees=&labels=idea&template=feature-request.md&title="
              target="_blank"
              rel="noopener noreferrer"
            >
              Suggest a new destination&nbsp;
              <FleetIcon name="external-link" />
            </a>
          </p>
        </InfoBanner>
        <form
          className={`${baseClass}__form`}
          onSubmit={onFormSubmit}
          autoComplete="off"
        >
          <InputField
            autofocus
            name="name"
            onChange={onInputChange}
            label="Jira site URL"
            placeholder="https://jira.example.com"
            value={name}
            error={errors.name}
          />
          <InputField
            autofocus
            name="name"
            onChange={onInputChange}
            label="Jira username"
            placeholder="name@example.com"
            value={name}
            error={errors.name}
            tooltip={
              "\
              This user must have “Create issues” for the project <br/> \
              in which the issues are created. \
            "
            }
          />
          <InputField
            autofocus
            name="name"
            onChange={onInputChange}
            label="Jira password"
            value={name}
            error={errors.name}
          />
          <InputField
            autofocus
            name="name"
            onChange={onInputChange}
            label="Jira project key"
            placeholder="JRAEXAMPLE"
            value={name}
            error={errors.name}
            tooltip={
              "\
              To find the Jira project key, head to your project in <br /> \
              Jira. Your project key is in URL. For example, in <br /> \
              “jira.example.com/projects/JRAEXAMPLE,” <br /> \
              “JRAEXAMPLE” is your project key. \
            "
            }
          />
          <div className={`${baseClass}__btn-wrap`}>
            <Button
              className={`${baseClass}__btn`}
              type="submit"
              variant="brand"
              disabled={name === ""}
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
      </>
    </Modal>
  );
};

export default CreateIntegrationModal;
