import React, { FormEvent, useState } from "react";
import ReactTooltip from "react-tooltip";

import {
  IJiraIntegration,
  IJiraIntegrationFormData,
  IJiraIntegrationFormErrors,
} from "interfaces/integration";

import Button from "components/buttons/Button";
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

interface IFormField {
  name: string;
  value: string;
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

  const { url, username, password, projectKey } = formData;

  const onInputChange = ({ name, value }: IFormField) => {
    setErrors({});
    setFormData({ ...formData, [name]: value });
  };

  // IntegrationForm component can be used to create a new jira integration or edit an existing jira integration so submitData will be assembled accordingly
  const createSubmitData = (): IJiraIntegration[] => {
    let jiraIntegrationSubmitData = integrations;

    if (integrationEditing) {
      // Edit existing integration
    } else {
      // Create new integration
      jiraIntegrationSubmitData = [
        ...jiraIntegrationSubmitData,
        {
          url,
          username,
          password,
          project_key: projectKey,
        },
      ];
    }

    return jiraIntegrationSubmitData;
  };

  const validateForm = (name: string) => {
    const validationErrors: IJiraIntegrationFormErrors = {};

    switch (name) {
      case "url":
        if (!url) {
          validationErrors.url = "Jira URL is required";
        }
        break;
      case "username":
        if (!username) {
          validationErrors.username = "Jira username is required";
        }
        break;
      case "password":
        if (!password) {
          validationErrors.password = "Jira password is required";
        }
        break;
      default:
        if (!projectKey) {
          validationErrors.projectKey = "Project Key is required";
        }
        break;
    }

    setErrors(validationErrors);
  };

  const onFormSubmit = (evt: FormEvent): void => {
    evt.preventDefault();

    return onSubmit(createSubmitData());
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
        parseTarget
        // onBlur={validateForm("url")}
        value={url}
        error={errors.url}
      />
      <InputField
        name="username"
        onChange={onInputChange}
        label="Jira username"
        placeholder="name@example.com"
        parseTarget
        // onBlur={validateForm("username")}
        value={username}
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
        parseTarget
        // onBlur={validateForm("password")}
        value={password}
        error={errors.password}
      />
      <InputField
        name="projectKey"
        onChange={onInputChange}
        label="Jira project key"
        placeholder="JRAEXAMPLE"
        parseTarget
        // onBlur={validateForm("projectKey")}
        value={projectKey}
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
        <div
          data-tip
          data-for="create-integration-button"
          data-tip-disable={
            !(
              formData.url === "" ||
              formData.username === "" ||
              formData.password === "" ||
              formData.projectKey === ""
            )
          }
        >
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
            Save
          </Button>
        </div>{" "}
        <ReactTooltip
          className={`create-integration-tooltip`}
          place="bottom"
          type="dark"
          effect="solid"
          backgroundColor="#3e4771"
          id="create-integration-button"
          data-html
        >
          <div
            className={`tooltip`}
            style={{ width: "152px", textAlign: "center" }}
          >
            All fields are required
          </div>
        </ReactTooltip>
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
