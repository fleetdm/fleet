import React, { FormEvent, useState, useEffect } from "react";
import ReactTooltip from "react-tooltip";

import {
  IIntegrationFormData,
  IIntegrationTableData,
  IIntegration,
  IIntegrations,
} from "interfaces/integration";

import Button from "components/buttons/Button";
// @ts-ignore
import InputField from "components/forms/fields/InputField";

const baseClass = "integration-form";

interface IIntegrationFormProps {
  onCancel: () => void;
  onSubmit: (
    untegrationSubmitData: IIntegration[],
    integrationDestination: string
  ) => void;
  integrationEditing?: IIntegrationTableData;
  integrations: IIntegrations;
  integrationEditingUrl?: string;
  integrationEditingUsername?: string;
  integrationEditingEmail?: string;
  integrationEditingApiToken?: string;
  integrationEditingProjectKey?: string;
  integrationEditingGroupId?: number;
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
    groupId: integrationEditingGroupId || 0,
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
  const createSubmitData = (): IIntegration[] => {
    let jiraIntegrationSubmitData = jiraIntegrations || [];
    let zendeskIntegrationSubmitData = zendeskIntegrations || [];

    console.log("jiraIntegrationSubmitData", jiraIntegrationSubmitData);
    console.log("zendeskIntegrationSubmitData", zendeskIntegrationSubmitData);

    if (integrationDestination === "jira") {
      if (
        integrationEditing &&
        integrationEditing.originalIndex &&
        integrationEditing.username
      ) {
        console.log("Edit existing jira integration");
        // Edit existing jira integration using array replacement
        jiraIntegrationSubmitData.splice(integrationEditing.originalIndex, 1, {
          url,
          username: username || "",
          api_token: apiToken,
          project_key: projectKey || "",
        });
      } else {
        console.log("Create new jira integration");
        // Create new jira integration at end of array
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
    }
    console.log("integrationEditing", integrationEditing);
    console.log(
      "integrationEditing.originalIndex",
      integrationEditing?.originalIndex
    );
    console.log("integrationEditing.email", integrationEditing?.email);
    if (
      integrationEditing &&
      integrationEditing.originalIndex &&
      integrationEditing.email
    ) {
      console.log("Edit existing zendesk jira integration");
      // Edit existing zendesk integration using array replacement
      zendeskIntegrationSubmitData.splice(integrationEditing.originalIndex, 1, {
        url,
        email: email || "",
        api_token: apiToken,
        group_id: groupId || 0,
      });
    } else {
      console.log("Create new zendesk integration");
      // Create new zendesk integration at end of array
      zendeskIntegrationSubmitData = [
        ...zendeskIntegrationSubmitData,
        {
          url,
          email: email || "",
          api_token: apiToken,
          group_id: parseInt(groupId as any) || 0,
        },
      ];
    }
    return zendeskIntegrationSubmitData;
  };

  const onFormSubmit = (evt: FormEvent): void => {
    evt.preventDefault();

    return onSubmit(createSubmitData(), integrationDestination);
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
          placeholder="12345678"
          type="number"
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
            !(integrationDestination === "jira"
              ? formData.url === "" ||
                formData.username === "" ||
                formData.apiToken === "" ||
                formData.projectKey === ""
              : formData.url === "" ||
                formData.email === "" ||
                formData.apiToken === "" ||
                formData.groupId === 0)
          }
        >
          <Button
            className={`${baseClass}__btn`}
            type="submit"
            variant="brand"
            disabled={
              integrationDestination === "jira"
                ? formData.url === "" ||
                  formData.username === "" ||
                  formData.apiToken === "" ||
                  formData.projectKey === ""
                : formData.url === "" ||
                  formData.email === "" ||
                  formData.apiToken === "" ||
                  formData.groupId === 0
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
