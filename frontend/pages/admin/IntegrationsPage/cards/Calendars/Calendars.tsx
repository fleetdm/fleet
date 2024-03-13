import React, { useState, useContext } from "react";
import { useQuery } from "react-query";

import { IConfig } from "interfaces/config";
import { NotificationContext } from "context/notification";
import { AppContext } from "context/app";
import configAPI from "services/entities/config";

// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Button from "components/buttons/Button";
import SectionHeader from "components/SectionHeader";
import CustomLink from "components/CustomLink";
import Spinner from "components/Spinner";
import Icon from "components/Icon";

import { ICalendarsFormErrors, IFormField } from "./constants";

const baseClass = "calendars-form";

const Calendars = (): JSX.Element => {
  const { renderFlash } = useContext(NotificationContext);

  const [formData, setFormData] = useState<any>({
    email: "",
    domain: "",
    privateKey: "",
  });

  const {
    data: appConfig,
    isLoading: isLoadingAppConfig,
    refetch: refetchConfig,
  } = useQuery<IConfig, Error, IConfig>(["config"], () => configAPI.loadAll(), {
    select: (data: IConfig) => data,
    onSuccess: (data) => {
      setFormData({
        email: data.integrations.google_calendar[0].email,
        domain: data.integrations.google_calendar[0].domain,
        privateKey: data.integrations.google_calendar[0].private_key,
      });
    },
  });

  const { email, domain, privateKey } = formData;

  const [isUpdatingSettings, setIsUpdatingSettings] = useState(false);
  const [formErrors, setFormErrors] = useState<ICalendarsFormErrors>({});

  const { isPremiumTier } = useContext(AppContext);

  const handleInputChange = ({ name, value }: IFormField) => {
    setFormData({ ...formData, [name]: value });
    setFormErrors({});
  };

  const validateForm = () => {
    const errors: ICalendarsFormErrors = {};
    if (!email) {
      errors.email = "Email must be present";
    }
    if (!domain) {
      errors.email = "Domain must be present";
    }
    if (!privateKey) {
      errors.privateKey = "Private key must be present";
    }

    setFormErrors(errors);
  };

  const onFormSubmit = (evt: React.MouseEvent<HTMLFormElement>) => {
    setIsUpdatingSettings(true);

    evt.preventDefault();

    // TODO: add validations

    // Formatting of API not UI
    const formDataToSubmit = {
      email: formData.email,
      domain: formData.domain,
      private_key: formData.privateKey,
    };

    // Updates integrations.google_calendar only
    const destination = {
      zendesk: appConfig?.integrations.zendesk,
      jira: appConfig?.integrations.jira,
      google_calendar: formDataToSubmit,
    };

    configAPI
      .update({ integrations: destination })
      .then(() => {
        renderFlash(
          "success",
          <>Successfully updated Google calendar settings</>
        );
        refetchConfig();
      })
      .catch( data: IApiError ) => {
        renderFlash(
          "error",
          <>
            Could not add <b>Google calendar integration</b>. Please try again.
          </>
        );
      })
      .finally(() => {
        setIsUpdatingSettings(false);
      });
  };

  const renderForm = () => {
    return isPremiumTier ? (
      <form onSubmit={onFormSubmit} autoComplete="off">
        <SectionHeader title="Calendars" />
        <p className={`${baseClass}__page-description`}>
          Connect Fleet to your Google Workspace service account to create
          calendar events for end users if their host fails policies.{" "}
          <CustomLink url="TODO" text="Learn more" newTab />
        </p>
        <InputField
          label="Email"
          onChange={handleInputChange}
          name="email"
          value={email}
          parseTarget
          tooltip={
            <>
              The email address for this Google
              <br /> Workspace service account.
            </>
          }
          placeholder="name@example.com"
        />
        <InputField
          label="Domain"
          onChange={handleInputChange}
          name="domain"
          value={domain}
          parseTarget
          tooltip={
            <>
              The Google Workspace domain this <br /> service account is
              associated with.
            </>
          }
          placeholder="example.com"
        />
        <InputField
          label="Private key"
          onChange={handleInputChange}
          name="private key"
          value={privateKey}
          parseTarget
          tooltip={
            <>
              The private key for this Google <br /> Workspace service account.
            </>
          }
          placeholder="•••••••••••••••••••••••••••••"
        />
        <Button
          type="submit"
          variant="brand"
          disabled={Object.keys(formErrors).length > 0}
          className="save-loading button-wrap"
          isLoading={isUpdatingSettings}
        >
          Save
        </Button>
      </form>
    ) : (
      // TODO: align icon
      <p>
        <Icon name="premium-feature" /> This feature is included in Fleet
        Premium. <CustomLink url="TODO" text="Learn more" newTab />
      </p>
    );
  };
  return (
    <div className={`${baseClass}`}>
      {isLoadingAppConfig ? <Spinner includeContainer={false} /> : renderForm()}
    </div>
  );
};

export default Calendars;
