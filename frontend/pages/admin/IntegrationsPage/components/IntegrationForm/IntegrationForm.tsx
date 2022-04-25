import React, { FormEvent, useState, useEffect } from "react";
import ReactTooltip from "react-tooltip";

import {
  IJiraIntegration,
  IZendeskIntegration,
  IIntegrationFormData,
  IIntegration,
  IIntegrations,
} from "interfaces/integration";

import Button from "components/buttons/Button";
// @ts-ignore
import InputField from "components/forms/fields/InputField";

const baseClass = "integration-form";

interface IIntegrationFormProps {
  onCancel: () => void;
  onSubmit: (jiraIntegrationSubmitData: IJiraIntegration[]) => void;
  integrationEditing?: IIntegrationFormData;
  integrations: IIntegrations;
  integrationEditingUrl?: string;
  integrationEditingUsername?: string;
  integrationEditingEmail?: string;
  integrationEditingApiToken?: string;
  integrationEditingProjectKey?: string;
  integrationEditingGroupId?: string;
  integrationEnableSoftwareVulnerabilities?: boolean;
  integrationEditingType?: string;
  destination?: string;
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
  integrationEditingUrl,
  integrationEditingUsername,
  integrationEditingEmail,
  integrationEditingApiToken,
  integrationEditingProjectKey,
  integrationEditingGroupId,
  integrationEnableSoftwareVulnerabilities,
  integrationEditingType,
  destination,
}: IIntegrationFormProps): JSX.Element => {
  console.log("integrationEditingType", integrationEditingType);
  const { jira: jiraIntegrations, zendesk: zendeskIntegrations } = integrations;
  const [formData, setFormData] = useState<IIntegrationFormData>({
    url: integrationEditingUrl || "",
    username: integrationEditingUsername || "",
    email: integrationEditingEmail || "",
    apiToken: integrationEditingApiToken || "",
    projectKey: integrationEditingProjectKey || "",
    groupId: integrationEditingGroupId || "",
    enableSoftwareVulnerabilities:
      integrationEnableSoftwareVulnerabilities || false,
  });
  const [integrationDestination, setIntegrationDestination] = useState<string>(
    integrationEditingType || destination || "jira"
  );

  useEffect(() => {
    setIntegrationDestination(destination || integrationEditingType || "jira");
  }, [destination, integrationEditingType]);

  const { url, username, email, apiToken, projectKey, groupId } = formData;

  const onInputChange = ({ name, value }: IFormField) => {
    setFormData({ ...formData, [name]: value });
  };

  // IntegrationForm component can be used to create a new jira integration or edit an existing jira integration so submitData will be assembled accordingly
  const createSubmitData = (): IJiraIntegration[] => {
    let jiraIntegrationSubmitData = jiraIntegrations;

    if (
      integrationEditing &&
      integrationEditing.originalIndex &&
      integrationEditing.username
    ) {
      // Edit existing integration using array replacement
      jiraIntegrationSubmitData.splice(integrationEditing.originalIndex, 1, {
        url,
        username: username || "",
        api_token: apiToken,
        project_key: projectKey || "",
      });
    } else {
      // Create new integration at end of array
      jiraIntegrationSubmitData = [
        ...jiraIntegrationSubmitData,
        {
          url,
          username: username || "",
          api_token: apiToken,
          project_key: projectKey || "",
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
        label="URL"
        placeholder="https://jira.example.com"
        parseTarget
        value={url}
      />
      {integrationDestination === "jira" ? (
        <InputField
          name="username"
          onChange={onInputChange}
          label="Username"
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
      ) : (
        <InputField
          name="email"
          onChange={onInputChange}
          label="Email"
          placeholder="name@example.com"
          parseTarget
          value={email}
        />
      )}
      <InputField
        name="apiToken"
        onChange={onInputChange}
        label="API token"
        parseTarget
        value={apiToken}
      />
      {integrationDestination === "jira" ? (
        <InputField
          name="projectKey"
          onChange={onInputChange}
          label="Project key"
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
      ) : (
        <InputField
          name="groupId"
          onChange={onInputChange}
          label="Group ID"
          placeholder="JRAEXAMPLE"
          parseTarget
          value={groupId}
          tooltip={
            "\
              To find the Zendesk group ID, select <b>Admin > <br /> \
              People > Groups</b>. Find the group and select it. <br /> \
              The group ID will appear in the search field. \
            "
          }
        />
      )}
      <div className={`${baseClass}__btn-wrap`}>
        <div
          data-tip
          data-for="create-integration-button"
          data-tip-disable={
            !(
              formData.url === "" ||
              formData.username === "" ||
              formData.apiToken === "" ||
              formData.projectKey === ""
            )
          }
        >
          <Button
            className={`${baseClass}__btn`}
            type="submit"
            variant="brand"
            disabled={
              destination === "jira"
                ? formData.url === "" ||
                  formData.username === "" ||
                  formData.apiToken === "" ||
                  formData.projectKey === ""
                : formData.url === "" ||
                  formData.email === "" ||
                  formData.apiToken === "" ||
                  formData.groupId === ""
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
            Complete all fields to save the integration.
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
