import React, { useState, useCallback, useEffect } from "react";

import Modal from "components/Modal";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Button from "components/buttons/Button";

const baseClass = "edit-team-modal";

export interface IEditTeamFormData {
  name: string;
}

interface IEditIntegrationModalProps {
  onCancel: () => void;
  onSubmit: (formData: IEditTeamFormData) => void;
  defaultName: string;
  backendValidators: { [key: string]: string };
}

const EditIntegrationModal = ({
  onCancel,
  onSubmit,
  defaultName,
  backendValidators,
}: IEditIntegrationModalProps): JSX.Element => {
  const [name, setName] = useState(defaultName);
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

  const onFormSubmit = (evt: React.MouseEvent<HTMLFormElement>) => {
    evt.preventDefault();
    onSubmit({ name });
  };

  return (
    <Modal title={"Edit integration"} onExit={onCancel} className={baseClass}>
      <>
        TODO: Copy form over from IntegrationForm.tsx which is also used on
        CreateIntegrationModal
      </>
    </Modal>
  );
};

export default EditIntegrationModal;
