import React, { useEffect, useState } from "react";

import {
  IIntegrationFormData,
  IIntegrationTableData,
  IIntegration,
  IZendeskJiraIntegrations,
  IIntegrationType,
} from "interfaces/integration";

import Button from "components/buttons/Button";
import InputField from "components/forms/fields/InputField";
import validUrl from "components/forms/validators/valid_url";
import useFormValidation, { IFormErrors } from "hooks/useFormValidation";

import Spinner from "components/Spinner";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";

const baseClass = "integration-form";

interface IIntegrationFormProps {
  onCancel: () => void;
  onSubmit: (
    integrationSubmitData: IIntegration[],
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
  gitOpsModeEnabled?: boolean;
}

const validateForm = (
  data: IIntegrationFormData,
  destination: string
): IFormErrors => {
  const errors: IFormErrors = {};

  if (!data.url) {
    errors.url = "Enter a URL";
  } else if (!validUrl({ url: data.url, protocols: ["https"] })) {
    errors.url = "Enter a valid HTTPS URL";
  }

  if (!data.apiToken) {
    errors.apiToken = "Enter an API token";
  }

  if (destination === "jira") {
    if (!data.username) {
      errors.username = "Enter a username";
    }
    if (!data.projectKey) {
      errors.projectKey = "Enter a project key";
    }
  } else {
    if (!data.email) {
      errors.email = "Enter your email";
    }
    if (!data.groupId) {
      errors.groupId = "Enter a group ID";
    }
  }

  return errors;
};

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
  gitOpsModeEnabled,
}: IIntegrationFormProps): JSX.Element => {
  const { jira: jiraIntegrations, zendesk: zendeskIntegrations } = integrations;

  const [integrationDestination, setIntegrationDestination] = useState(
    integrationEditingType || destination || "jira"
  );

  useEffect(() => {
    setIntegrationDestination(destination || integrationEditingType || "jira");
  }, [destination, integrationEditingType]);

  const {
    formData,
    setField,
    getError,
    clearFieldError,
    validateField,
    handleSubmit,
    isSubmitting,
  } = useFormValidation<IIntegrationFormData>({
    initialFormData: {
      url: integrationEditingUrl || "",
      username: integrationEditingUsername || "",
      email: integrationEditingEmail || "",
      apiToken: integrationEditingApiToken || "",
      projectKey: integrationEditingProjectKey || "",
      groupId: integrationEditingGroupId || 0,
      enableSoftwareVulnerabilities:
        integrationEnableSoftwareVulnerabilities || false,
    },
    validate: (data) => validateForm(data, integrationDestination),
    isSubmitting: testingConnection,
    skipTrim: ["apiToken"],
  });

  // IntegrationForm component can be used to create a new integration or edit
  // an existing integration, so submitData will be assembled accordingly.
  const createSubmitData = (data: IIntegrationFormData): IIntegration[] => {
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
          url: data.url,
          username: data.username || "",
          api_token: data.apiToken,
          project_key: data.projectKey || "",
        });
      } else {
        // Create new jira integration at end of array
        jiraIntegrationSubmitData = [
          ...jiraIntegrationSubmitData,
          {
            url: data.url,
            username: data.username || "",
            api_token: data.apiToken,
            project_key: data.projectKey || "",
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
        url: data.url,
        email: data.email || "",
        api_token: data.apiToken,
        group_id: Number(data.groupId) || 0,
      });
    } else {
      // Create new zendesk integration at end of array
      zendeskIntegrationSubmitData = [
        ...zendeskIntegrationSubmitData,
        {
          url: data.url,
          email: data.email || "",
          api_token: data.apiToken,
          group_id: Number(data.groupId) || 0,
        },
      ];
    }
    return zendeskIntegrationSubmitData;
  };

  const onValidSubmit = (data: IIntegrationFormData) => {
    onSubmit(createSubmitData(data), integrationDestination);
  };

  if (testingConnection) {
    return (
      <div className={`${baseClass}__testing-connection`}>
        <b>Testing connection</b>
        <Spinner />
      </div>
    );
  }

  return (
    <form
      className={`${baseClass}__form`}
      onSubmit={handleSubmit(onValidSubmit)}
      autoComplete="off"
      noValidate
    >
      <InputField
        autofocus
        name="url"
        label="URL"
        placeholder={
          integrationDestination === "jira"
            ? "https://example.atlassian.net"
            : "https://example.zendesk.com"
        }
        value={formData.url}
        onChange={(value: string) => setField("url", value)}
        onFocus={() => clearFieldError("url")}
        onBlur={() => validateField("url")}
        error={getError("url")}
        disabled={gitOpsModeEnabled || isSubmitting}
      />
      {integrationDestination === "jira" ? (
        <InputField
          name="username"
          label="Username"
          placeholder="name@example.com"
          value={formData.username || ""}
          onChange={(value: string) => setField("username", value)}
          onFocus={() => clearFieldError("username")}
          onBlur={() => validateField("username")}
          error={getError("username")}
          disabled={gitOpsModeEnabled || isSubmitting}
        />
      ) : (
        <InputField
          name="email"
          label="Email"
          placeholder="name@example.com"
          type="email"
          value={formData.email || ""}
          onChange={(value: string) => setField("email", value)}
          onFocus={() => clearFieldError("email")}
          onBlur={() => validateField("email")}
          error={getError("email")}
          disabled={gitOpsModeEnabled || isSubmitting}
        />
      )}
      <InputField
        name="apiToken"
        label="API token"
        value={formData.apiToken}
        onChange={(value: string) => setField("apiToken", value)}
        onFocus={() => clearFieldError("apiToken")}
        onBlur={() => validateField("apiToken")}
        error={getError("apiToken")}
        disabled={gitOpsModeEnabled || isSubmitting}
      />
      {integrationDestination === "jira" ? (
        <InputField
          name="projectKey"
          label="Project key"
          placeholder="JRAEXAMPLE"
          value={formData.projectKey || ""}
          onChange={(value: string) => setField("projectKey", value)}
          onFocus={() => clearFieldError("projectKey")}
          onBlur={() => validateField("projectKey")}
          error={getError("projectKey")}
          disabled={gitOpsModeEnabled || isSubmitting}
          tooltip={
            <>
              To find the Jira project key, head to your project in <br />
              Jira. Your project key is located in the URL. For example, in{" "}
              <br />
              &ldquo;jira.example.com/projects/JRAEXAMPLE,&rdquo; <br />
              &ldquo;JRAEXAMPLE&rdquo; is your project key.
            </>
          }
        />
      ) : (
        <InputField
          name="groupId"
          label="Group ID"
          placeholder="28134038"
          type="number"
          value={formData.groupId ? formData.groupId : null}
          onChange={(value: string) =>
            setField("groupId", value ? Number(value) : 0)
          }
          onFocus={() => clearFieldError("groupId")}
          onBlur={() => validateField("groupId")}
          error={getError("groupId")}
          disabled={gitOpsModeEnabled || isSubmitting}
          tooltip={
            <>
              To find the Zendesk group ID, select{" "}
              <strong>Admin &gt; People &gt; Groups</strong>. Find the group and
              select it. The group ID will appear in the search field.
            </>
          }
        />
      )}
      <div className="modal-cta-wrap">
        <GitOpsModeTooltipWrapper
          tipOffset={8}
          renderChildren={(disableChildren) => (
            <Button
              type="submit"
              disabled={disableChildren || isSubmitting}
              isLoading={isSubmitting}
            >
              Add
            </Button>
          )}
        />
        <Button onClick={onCancel} variant="secondary">
          Cancel
        </Button>
      </div>
    </form>
  );
};

export default IntegrationForm;
