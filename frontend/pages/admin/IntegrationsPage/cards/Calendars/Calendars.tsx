import React, { useState, useContext, useCallback } from "react";
import { useQuery } from "react-query";

import { IConfig } from "interfaces/config";
import { NotificationContext } from "context/notification";
import { AppContext } from "context/app";
import configAPI from "services/entities/config";
// @ts-ignore
import { stringToClipboard } from "utilities/copy_text";
import paths from "router/paths";

// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Button from "components/buttons/Button";
import SectionHeader from "components/SectionHeader";
import CustomLink from "components/CustomLink";
import Spinner from "components/Spinner";
import DataError from "components/DataError";
import PremiumFeatureMessage from "components/PremiumFeatureMessage/PremiumFeatureMessage";
import Icon from "components/Icon";
import Card from "components/Card";

const CREATING_SERVICE_ACCOUNT =
  "https://www.fleetdm.com/learn-more-about/creating-service-accounts";
const GOOGLE_WORKSPACE_DOMAINS =
  "https://www.fleetdm.com/learn-more-about/google-workspace-domains";
const DOMAIN_WIDE_DELEGATION =
  "https://www.fleetdm.com/learn-more-about/domain-wide-delegation";
const ENABLING_CALENDAR_API =
  "https://www.fleetdm.com/learn-more-about/enabling-calendar-api";
const OAUTH_SCOPES =
  "https://www.googleapis.com/auth/calendar.events,https://www.googleapis.com/auth/calendar.settings.readonly";

const API_KEY_JSON_PLACEHOLDER = `{
  "type": "service_account",
  "project_id": "fleet-in-your-calendar",
  "private_key_id": "<private key id>",
  "private_key": "-----BEGIN PRIVATE KEY----\\n<private key>\\n-----END PRIVATE KEY-----\\n",
  "client_email": "fleet-calendar-events@fleet-in-your-calendar.iam.gserviceaccount.com",
  "client_id": "<client id>",
  "auth_uri": "https://accounts.google.com/o/oauth2/auth",
  "token_uri": "https://oauth2.googleapis.com/token",
  "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
  "client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/fleet-calendar-events%40fleet-in-your-calendar.iam.gserviceaccount.com",
  "universe_domain": "googleapis.com"
}`;

interface IFormField {
  name: string;
  value: string | boolean | number;
}

interface ICalendarsFormErrors {
  domain?: string | null;
  apiKeyJson?: string | null;
}

interface ICalendarsFormData {
  domain?: string;
  apiKeyJson?: string;
}

// Used to surface error.message in UI of unknown error type
type ErrorWithMessage = {
  message: string;
  [key: string]: unknown;
};

const isErrorWithMessage = (error: unknown): error is ErrorWithMessage => {
  return (error as ErrorWithMessage).message !== undefined;
};

const baseClass = "calendars-integration";

const Calendars = (): JSX.Element => {
  const { renderFlash } = useContext(NotificationContext);
  const { isPremiumTier } = useContext(AppContext);

  const [formData, setFormData] = useState<ICalendarsFormData>({
    domain: "",
    apiKeyJson: "",
  });
  const [isUpdatingSettings, setIsUpdatingSettings] = useState(false);
  const [formErrors, setFormErrors] = useState<ICalendarsFormErrors>({});
  const [copyMessage, setCopyMessage] = useState<string>("");

  const {
    isLoading: isLoadingAppConfig,
    refetch: refetchConfig,
    error: errorAppConfig,
  } = useQuery<IConfig, Error, IConfig>(["config"], () => configAPI.loadAll(), {
    select: (data: IConfig) => data,
    onSuccess: (data) => {
      if (data.integrations.google_calendar) {
        setFormData({
          domain: data.integrations.google_calendar[0].domain,
          // Formats string for better UI readability
          apiKeyJson: JSON.stringify(
            data.integrations.google_calendar[0].api_key_json,
            null,
            "\t"
          ),
        });
      }
    },
  });

  const { apiKeyJson, domain } = formData;

  const validateForm = (curFormData: ICalendarsFormData) => {
    const errors: ICalendarsFormErrors = {};

    // Must set all keys or no keys at all
    if (!curFormData.apiKeyJson && !!curFormData.domain) {
      errors.apiKeyJson = "API key JSON must be completed";
    }
    if (!curFormData.domain && !!curFormData.apiKeyJson) {
      errors.domain = "Domain must be completed";
    }
    if (curFormData.apiKeyJson) {
      try {
        JSON.parse(curFormData.apiKeyJson);
      } catch (e: unknown) {
        if (isErrorWithMessage(e)) {
          errors.apiKeyJson = e.message.toString();
        } else {
          throw e;
        }
      }
    }
    return errors;
  };

  const onInputChange = useCallback(
    ({ name, value }: IFormField) => {
      const newFormData = { ...formData, [name]: value };
      setFormData(newFormData);
      setFormErrors(validateForm(newFormData));
    },
    [formData]
  );

  const onFormSubmit = async (evt: React.MouseEvent<HTMLFormElement>) => {
    setIsUpdatingSettings(true);

    evt.preventDefault();

    // Format for API
    const formDataToSubmit =
      formData.apiKeyJson === "" && formData.domain === ""
        ? [] // Send empty array if no keys are set
        : [
            {
              domain: formData.domain,
              api_key_json:
                (formData.apiKeyJson && JSON.parse(formData.apiKeyJson)) ||
                null,
            },
          ];

    // Update integrations.google_calendar only
    const destination = {
      google_calendar: formDataToSubmit,
    };

    try {
      await configAPI.update({ integrations: destination });
      renderFlash(
        "success",
        "Successfully saved calendar integration settings."
      );
      refetchConfig();
    } catch (e) {
      renderFlash("error", "Could not save calendar integration settings.");
    } finally {
      setIsUpdatingSettings(false);
    }
  };

  const renderOauthLabel = () => {
    const onCopyOauthScopes = (evt: React.MouseEvent) => {
      evt.preventDefault();

      stringToClipboard(OAUTH_SCOPES)
        .then(() => setCopyMessage(() => "Copied!"))
        .catch(() => setCopyMessage(() => "Copy failed"));

      // Clear message after 1 second
      setTimeout(() => setCopyMessage(() => ""), 1000);

      return false;
    };

    return (
      <span className={`${baseClass}__oauth-scopes-copy-icon-wrapper`}>
        <Button
          variant="unstyled"
          className={`${baseClass}__oauth-scopes-copy-icon`}
          onClick={onCopyOauthScopes}
        >
          <Icon name="copy" />
        </Button>
        {copyMessage && (
          <span className={`${baseClass}__copy-message`}>{copyMessage}</span>
        )}
      </span>
    );
  };

  const renderForm = () => {
    return (
      <>
        <SectionHeader title="Calendars" />
        <p className={`${baseClass}__page-description`}>
          To create calendar events for end users with failing policies,
          you&apos;ll need to configure a dedicated Google Workspace service
          account.
        </p>
        <div className={`${baseClass}__section-instructions`}>
          <p>
            1. Go to the <b>Service Accounts</b> page in Google Cloud Platform.{" "}
            <CustomLink
              text="View page"
              url={CREATING_SERVICE_ACCOUNT}
              newTab
            />
          </p>
          <p>
            2. Create a new project for your service account.
            <ul>
              <li>
                Click <b>Create project</b>.
              </li>
              <li>
                Enter &quot;Fleet calendar events&quot; as the project name.
              </li>
              <li>
                For &quot;Organization&quot; and &quot;Location&quot;, select
                your calendar&apos;s organization.
              </li>
              <li>
                Click <b>Create</b>.
              </li>
            </ul>
          </p>

          <p>
            3. Create the service account.
            <ul>
              <li>
                Click <b>Create service account</b>.
              </li>
              <li>
                Set the service account name to &quot;Fleet calendar
                events&quot;.
              </li>
              <li>
                Set the service account ID to &quot;fleet-calendar-events&quot;.
              </li>
              <li>
                Click <b>Create and continue</b>.
              </li>
              <li>
                Click <b>Done</b> at the bottom of the form. (No need to
                complete the optional steps.)
              </li>
            </ul>
          </p>
          <p>
            4. Create an API key.{" "}
            <ul>
              <li>
                Click the <b>Actions</b> menu for your new service account.
              </li>
              <li>
                Select <b>Manage keys</b>.
              </li>
              <li>
                Click <b>Add key &gt; Create new key</b>.
              </li>
              <li>Select the JSON key type.</li>
              <li>
                Click <b>Create</b> to create the key & download a JSON file.
              </li>
            </ul>
          </p>
          <p className={`${baseClass}__configuration`}>
            5. Configure your service account integration in Fleet using the
            form below.
            <ul>
              <li>
                Paste the full contents of the JSON file downloaded when
                creating your service account API key.
              </li>
              <li>
                Set your primary domain. (If the end user is signed into
                multiple Google accounts, this will be used to identify their
                work calendar.)
              </li>
              <li>
                Save your changes.
                <Card>
                  <form onSubmit={onFormSubmit} autoComplete="off">
                    <InputField
                      label="API key JSON"
                      onChange={onInputChange}
                      name="apiKeyJson"
                      value={apiKeyJson}
                      parseTarget
                      type="textarea"
                      placeholder={API_KEY_JSON_PLACEHOLDER}
                      ignore1password
                      inputClassName={`${baseClass}__api-key-json`}
                      error={formErrors.apiKeyJson}
                    />
                    <InputField
                      label="Primary domain"
                      onChange={onInputChange}
                      name="domain"
                      value={domain}
                      parseTarget
                      placeholder="example.com"
                      helpText={
                        <>
                          You can find your primary domain in Google Workspace{" "}
                          <CustomLink
                            url={GOOGLE_WORKSPACE_DOMAINS}
                            text="here"
                            newTab
                          />
                        </>
                      }
                      error={formErrors.domain}
                    />
                    <Button
                      type="submit"
                      variant="brand"
                      disabled={Object.keys(formErrors).length > 0}
                      className="save-loading"
                      isLoading={isUpdatingSettings}
                    >
                      Save
                    </Button>
                  </form>
                </Card>
              </li>
            </ul>
          </p>
          <p>
            6. Authorize the service account via domain-wide delegation.
            <ul>
              <li>
                In Google Workspace, go to{" "}
                <b>
                  Security &gt; Access and data control &gt; API controls &gt;
                  Manage Domain Wide Delegation
                </b>
                .{" "}
                <CustomLink
                  url={DOMAIN_WIDE_DELEGATION}
                  text="View page"
                  newTab
                />
              </li>
              <li>
                Under <b>API clients</b>, click <b>Add new</b>.
              </li>
              <li>
                Enter the client ID for the service account. You can find this
                in your downloaded API key JSON file (
                <span className={`${baseClass}__code`}>client_id</span>
                ), or under <b>Advanced Settings</b> when viewing the service
                account.
              </li>
              <li>
                For the OAuth scopes, paste the following value:
                <InputField
                  readOnly
                  inputWrapperClass={`${baseClass}__oauth-scopes`}
                  name="oauth-scopes"
                  label={renderOauthLabel()}
                  type="textarea"
                  value={OAUTH_SCOPES}
                />
              </li>
              <li>
                Click <b>Authorize</b>.
              </li>
            </ul>
          </p>
          <p>
            7. Enable the Google Calendar API.
            <ul>
              <li>
                In the Google Cloud console API library, go to the Google
                Calendar API.{" "}
                <CustomLink
                  url={ENABLING_CALENDAR_API}
                  text="View page"
                  newTab
                />
              </li>
              <li>
                Make sure the &quot;Fleet calendar events&quot; project is
                selected at the top of the page.
              </li>
              <li>
                Click <b>Enable</b>.
              </li>
            </ul>
          </p>
          <p>
            Now head over to{" "}
            <CustomLink
              url={paths.MANAGE_POLICIES}
              text="Policies &gt; Manage automations"
            />{" "}
            to finish setup.
          </p>
        </div>
      </>
    );
  };

  if (!isPremiumTier) return <PremiumFeatureMessage />;

  if (isLoadingAppConfig) {
    <div className={baseClass}>
      <Spinner includeContainer={false} />
    </div>;
  }

  if (errorAppConfig) {
    return <DataError />;
  }

  return <div className={baseClass}>{renderForm()}</div>;
};

export default Calendars;
