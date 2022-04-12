import React, { FormEvent, useState } from "react";
import ReactTooltip from "react-tooltip";

import {
  IJiraIntegration,
  IJiraIntegrationFormData,
  IJiraIntegrationIndexed,
} from "interfaces/integration";

import Button from "components/buttons/Button";
// @ts-ignore
import InputField from "components/forms/fields/InputField";

const baseClass = "integration-form";

interface IIntegrationFormProps {
  onCancel: () => void;
  onSubmit: (jiraIntegrationSubmitData: IJiraIntegration[]) => void;
  integrationEditing?: IJiraIntegrationIndexed;
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
  integrations,
}: IIntegrationFormProps): JSX.Element => {
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
    setFormData({ ...formData, [name]: value });
  };

  // IntegrationForm component can be used to create a new jira integration or edit an existing jira integration so submitData will be assembled accordingly
  const createSubmitData = (): IJiraIntegration[] => {
    let jiraIntegrationSubmitData = integrations;

    if (integrationEditing) {
      // Edit existing integration using array replacement
      jiraIntegrationSubmitData.splice(integrationEditing.index, 1, {
        url,
        username,
        password,
        project_key: projectKey,
      });
    } else {
      // Create new integration at end of array
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
        value={url}
      />
      <InputField
        name="username"
        onChange={onInputChange}
        label="Jira username"
        placeholder="name@example.com"
        parseTarget
        value={username}
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
        value={password}
      />
      <InputField
        name="projectKey"
        onChange={onInputChange}
        label="Jira project key"
        placeholder="JRAEXAMPLE"
        parseTarget
        value={projectKey}
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
            Complete all fields to save the integration
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
