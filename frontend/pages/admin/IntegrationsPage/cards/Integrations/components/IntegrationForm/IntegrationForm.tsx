import React, { FormEvent, useState, useEffect } from "react";
import ReactTooltip from "react-tooltip";

import {
  IIntegrationFormData,
  IIntegrationTableData,
  IIntegration,
  IZendeskJiraIntegrations,
  IIntegrationType,
} from "interfaces/integration";

import Button from "components/buttons/Button";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import validUrl from "components/forms/validators/valid_url";

import Spinner from "components/Spinner";
import { COLORS } from "styles/var/colors";

const baseClass = "integration-form";

interface IIntegrationFormProps {
  onCancel: () => void;
  onSubmit: (
    untegrationSubmitData: IIntegration[],
    integrationDestination: string
  ) => void;
  integrationEditing?: IIntegrationTableData;
  integrations: IZendeskJiraIntegrations;
  integrationEditingUrl?: string;
  integrationEditingUsername?: string;
  integrationEditingEmail?: string;
  integrationEditingApiToken?: string;
  integrationEditingProjectKey?: string;
  integrationEditingGroupId?: number;
  integrationEnableSoftwareVulnerabilities?: boolean;
  integrationEditingType?: IIntegrationType;
  destination?: string;
  testingConnection?: boolean;
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
  testingConnection,
}: IIntegrationFormProps): JSX.Element => {
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
  const [integrationDestination, setIntegrationDestination] = useState(
    integrationEditingType || destination || "jira"
  );
  const [urlError, setUrlError] = useState<string | null>(null);

  useEffect(() => {
    setIntegrationDestination(destination || integrationEditingType || "jira");
  }, [destination, integrationEditingType]);

  const { url, username, email, apiToken, projectKey, groupId } = formData;

  const onInputChange = ({ name, value }: IFormField) => {
    setFormData({ ...formData, [name]: value });
  };

  const validateForm = () => {
    let error = null;

    if (url && !validUrl({ url, protocols: ["https"] })) {
      error = `${url} is not a valid HTTPS URL`;
    }

    setUrlError(error);
  };

  // IntegrationForm component can be used to create a new integration or edit an existing integration so submitData will be assembled accordingly
  const createSubmitData = (): IIntegration[] => {
    let jiraIntegrationSubmitData = jiraIntegrations || [];
    let zendeskIntegrationSubmitData = zendeskIntegrations || [];

    // Editing through UI is temporarily deprecated in 4.14
    if (integrationDestination === "jira") {
      if (
        integrationEditing &&
        (integrationEditing.originalIndex ||
          integrationEditing.originalIndex === 0) &&
        integrationEditing.username
      ) {
        // Edit existing jira integration using array replacement
        jiraIntegrationSubmitData.splice(integrationEditing.originalIndex, 1, {
          url,
          username: username || "",
          api_token: apiToken,
          project_key: projectKey || "",
        });
      } else {
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
    if (
      integrationEditing &&
      (integrationEditing.originalIndex ||
        integrationEditing.originalIndex === 0) &&
      integrationEditing.email
    ) {
      // Edit existing zendesk integration using array replacement
      zendeskIntegrationSubmitData.splice(integrationEditing.originalIndex, 1, {
        url,
        email: email || "",
        api_token: apiToken,
        group_id: groupId || 0,
      });
    } else {
      // Create new zendesk integration at end of array
      zendeskIntegrationSubmitData = [
        ...zendeskIntegrationSubmitData,
        {
          url,
          email: email || "",
          api_token: apiToken,
          group_id: parseInt(groupId as any, 10) || 0,
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
    <>
      {testingConnection ? (
        <div className={`${baseClass}__testing-connection`}>
          <b>Testing connection</b>
          <Spinner />
        </div>
      ) : (
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
            placeholder={
              integrationDestination === "jira"
                ? "https://example.atlassian.net"
                : "https://example.zendesk.com"
            }
            parseTarget
            value={url}
            error={urlError}
            onBlur={validateForm}
          />
          {integrationDestination === "jira" ? (
            <InputField
              name="username"
              onChange={onInputChange}
              label="Username"
              placeholder="name@example.com"
              parseTarget
              value={username}
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
                <>
                  To find the Jira project key, head to your project in <br />
                  Jira. Your project key is located in the URL. For example, in{" "}
                  <br />
                  “jira.example.com/projects/JRAEXAMPLE,” <br />
                  “JRAEXAMPLE” is your project key.
                </>
              }
            />
          ) : (
            <InputField
              name="groupId"
              onChange={onInputChange}
              label="Group ID"
              placeholder="28134038"
              type="number"
              parseTarget
              value={groupId === 0 ? null : groupId}
              tooltip={
                <>
                  To find the Zendesk group ID, select{" "}
                  <b>
                    Admin &gt; <br />
                    People &gt; Groups
                  </b>
                  . Find the group and select it. <br />
                  The group ID will appear in the search field.
                </>
              }
            />
          )}
          <div className="modal-cta-wrap">
            <div
              data-tip
              data-for="add-integration-button"
              data-tip-disable={
                !(integrationDestination === "jira"
                  ? formData.url === "" ||
                    formData.url.slice(0, 8) !== "https://" ||
                    formData.username === "" ||
                    formData.apiToken === "" ||
                    formData.projectKey === ""
                  : formData.url === "" ||
                    formData.url.slice(0, 8) !== "https://" ||
                    formData.email === "" ||
                    formData.apiToken === "" ||
                    formData.groupId === 0)
              }
              className="tooltip"
            >
              <Button
                type="submit"
                variant="brand"
                disabled={
                  integrationDestination === "jira"
                    ? formData.url === "" ||
                      formData.url.slice(0, 8) !== "https://" ||
                      formData.username === "" ||
                      formData.apiToken === "" ||
                      formData.projectKey === ""
                    : formData.url === "" ||
                      formData.url.slice(0, 8) !== "https://" ||
                      formData.email === "" ||
                      formData.apiToken === "" ||
                      formData.groupId === 0
                }
              >
                Save
              </Button>
            </div>
            <ReactTooltip
              className={`add-integration-tooltip`}
              place="bottom"
              effect="solid"
              backgroundColor={COLORS["tooltip-bg"]}
              id="add-integration-button"
              data-html
            >
              <>
                Complete all fields to save <br /> the integration.
              </>
            </ReactTooltip>
            <Button onClick={onCancel} variant="inverse">
              Cancel
            </Button>
          </div>
        </form>
      )}
    </>
  );
};

export default IntegrationForm;
