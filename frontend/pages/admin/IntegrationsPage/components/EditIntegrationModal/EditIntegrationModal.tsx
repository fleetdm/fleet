import React, { useState, useCallback, useEffect } from "react";

import Modal from "components/Modal";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Button from "components/buttons/Button";
import { IConfig } from "interfaces/config";
import { IJiraIntegration } from "interfaces/integration";

const baseClass = "edit-team-modal";

interface IEditIntegrationModalProps {
  onCancel: () => void;
  onSubmit: (formData: IJiraIntegration) => void;
  defaultName: string;
  backendValidators: { [key: string]: string };
}

const EditIntegrationModal = ({
  onCancel,
  onSubmit,
  defaultName,
  backendValidators,
}: IEditIntegrationModalProps): JSX.Element => {
  const [url, setURL] = useState<string>("");
  const [username, setUsername] = useState<string>("");
  const [password, setPassword] = useState<string>("");
  const [projectKey, setProjectKey] = useState<string>("");
  const [
    enableSoftwareVulnerabilities,
    setEnableSoftwareVulnerabilities,
  ] = useState<boolean>(false);

  const [errors, setErrors] = useState<{ [key: string]: string }>(
    backendValidators
  );

  // FIX ALL LATER!!!
  useEffect(() => {
    setErrors(backendValidators);
  }, [backendValidators]);

  const onInputChange = useCallback(
    (value: string) => {
      setURL(value);
      setErrors({});
    },
    [setURL]
  );

  const onFormSubmit = (evt: React.MouseEvent<HTMLFormElement>) => {
    evt.preventDefault();
    onSubmit({
      url: url,
      username: username,
      password: password,
      project_key: projectKey,
      enable_software_vulnerabilities: enableSoftwareVulnerabilities,
    });
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
