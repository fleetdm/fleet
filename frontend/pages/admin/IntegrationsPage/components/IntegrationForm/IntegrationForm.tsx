import React, { FormEvent, useState, useEffect } from "react";
import { useDispatch } from "react-redux";

import {
  IJiraIntegration,
  IJiraIntegrationFormData,
  IJiraIntegrationFormErrors,
} from "interfaces/integration";
import { IUserFormErrors } from "interfaces/user";
// ignore TS error for now until these are rewritten in ts.
import Button from "components/buttons/Button";
import validatePresence from "components/forms/validators/validate_presence";
// @ts-ignore
import InputField from "components/forms/fields/InputField";

const baseClass = "integration-form";

interface IIntegrationFormProps {
  onCancel: () => void;
  onSubmit: (jiraIntegrationSubmitData: IJiraIntegration[]) => void;
  serverErrors?: { base: string; email: string }; // "server" because this form does its own client validation
  createOrEditIntegrationErrors: IJiraIntegrationFormErrors;
  integrationEditing?: IJiraIntegration;
  integrations: IJiraIntegration[];
}

const IntegrationForm = ({
  onCancel,
  onSubmit,
  integrationEditing,
  serverErrors,
  createOrEditIntegrationErrors,
  integrations,
}: IIntegrationFormProps): JSX.Element => {
  const [errors, setErrors] = useState<any>(createOrEditIntegrationErrors);
  const [formData, setFormData] = useState<IJiraIntegrationFormData>({
    url: integrationEditing?.url || "",
    username: integrationEditing?.username || "",
    password: integrationEditing?.password || "",
    projectKey: integrationEditing?.project_key || "",
    enableSoftwareVulnerabilities:
      integrationEditing?.enable_software_vulnerabilities || false,
  });

  useEffect(() => {
    setErrors(createOrEditIntegrationErrors);
  }, [createOrEditIntegrationErrors]);

  const onInputChange = (formField: string): ((value: string) => void) => {
    return (value: string) => {
      setErrors({
        ...errors,
        [formField]: null,
      });
      setFormData({
        ...formData,
        [formField]: value,
      });
    };
  };

  // IntegrationForm component can be used to create a new jira integration or edit an existing jira integration so submitData will be assembled accordingly
  const createSubmitData = (): IJiraIntegration[] => {
    let jiraIntegrationSubmitData = integrations;

    if (!integrationEditing) {
      // add to the jira array
      jiraIntegrationSubmitData = [
        ...jiraIntegrationSubmitData,
        {
          url: formData.url,
          username: formData.username,
          password: formData.password,
          project_key: formData.projectKey,
        },
      ];
    } else {
      // modify the array
    }

    return jiraIntegrationSubmitData;
  };

  const validate = (): boolean => {
    if (!validatePresence(formData.url)) {
      setErrors({
        ...errors,
        url: "This field is required",
      });

      return false;
    }

    if (!validatePresence(formData.username)) {
      setErrors({
        ...errors,
        username: "This field is required",
      });

      return false;
    }
    if (!validatePresence(formData.password)) {
      setErrors({
        ...errors,
        password: "This field is required",
      });

      return false;
    }
    if (!validatePresence(formData.projectKey)) {
      setErrors({
        ...errors,
        projectKey: "This field is required",
      });

      return false;
    }
    return true;
  };

  const onFormSubmit = (evt: FormEvent): void => {
    evt.preventDefault();
    const valid = validate();
    if (valid) {
      return onSubmit(createSubmitData());
    }
  };

  return (
    <form
      className={`${baseClass}__form`}
      onSubmit={onFormSubmit}
      autoComplete="off"
    >
      <InputField
        autofocus
        name="url"
        onChange={onInputChange}
        label="Jira site URL"
        placeholder="https://jira.example.com"
        value={formData.url}
        error={errors.url}
      />
      <InputField
        name="username"
        onChange={onInputChange}
        label="Jira username"
        placeholder="name@example.com"
        value={formData.username}
        error={errors.username}
        tooltip={
          "\
              This user must have “Create issues” for the project <br/> \
              in which the issues are created. \
            "
        }
      />
      <InputField
        name="password"
        onChange={onInputChange}
        label="Jira password"
        value={formData.password}
        error={errors.password}
      />
      <InputField
        name="projectKey"
        onChange={onInputChange}
        label="Jira project key"
        placeholder="JRAEXAMPLE"
        value={formData.projectKey}
        error={errors.projectKey}
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
          disabled={
            formData.url === "" ||
            formData.username === "" ||
            formData.password === "" ||
            formData.projectKey === ""
          }
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
  );
};

export default IntegrationForm;
