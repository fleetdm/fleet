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
import DataError from "components/DataError";
import PremiumFeatureMessage from "components/PremiumFeatureMessage/PremiumFeatureMessage";

import {
  ICalendarsFormErrors,
  IFormField,
  LEARN_MORE_CALENDARS,
} from "./constants";

const baseClass = "calendars-form";

const Calendars = (): JSX.Element => {
  const { renderFlash } = useContext(NotificationContext);
  const { isPremiumTier } = useContext(AppContext);

  const [formData, setFormData] = useState({
    email: "",
    domain: "",
    privateKey: "",
  });
  const [isUpdatingSettings, setIsUpdatingSettings] = useState(false);
  const [formErrors, setFormErrors] = useState<ICalendarsFormErrors>({});

  const {
    data: appConfig,
    isLoading: isLoadingAppConfig,
    refetch: refetchConfig,
    error: errorAppConfig,
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

  const handleInputChange = ({ name, value }: IFormField) => {
    setFormData({ ...formData, [name]: value });
    setFormErrors({});
  };

  const validateForm = () => {
    const errors: ICalendarsFormErrors = {};

    // Must set all keys or no keys at all
    if (!email && (!!domain || !!privateKey)) {
      errors.email = "Email must be present";
    }
    if (!domain && (!!email || !!privateKey)) {
      errors.email = "Domain must be present";
    }
    if (!privateKey && (!!email || !!domain)) {
      errors.privateKey = "Private key must be present";
    }

    setFormErrors(errors);
  };

  const onFormSubmit = (evt: React.MouseEvent<HTMLFormElement>) => {
    setIsUpdatingSettings(true);

    evt.preventDefault();

    // Format for API
    const formDataToSubmit =
      formData.email === "" &&
      formData.domain === "" &&
      formData.privateKey === ""
        ? null // Send null if no keys are set
        : [
            {
              email: formData.email,
              domain: formData.domain,
              private_key: formData.privateKey,
            },
          ];

    // Update integrations.google_calendar only
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
      .catch(() => {
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
    return (
      <>
        <SectionHeader title="Calendars" />
        <form onSubmit={onFormSubmit} autoComplete="off">
          <p className={`${baseClass}__page-description`}>
            Connect Fleet to your Google Workspace service account to create
            calendar events for end users if their host fails policies.{" "}
            <CustomLink url={LEARN_MORE_CALENDARS} text="Learn more" newTab />
          </p>
          <InputField
            label="Email"
            onChange={handleInputChange}
            name="email"
            value={email}
            parseTarget
            onBlur={validateForm}
            tooltip={
              <>
                The email address for this Google
                <br /> Workspace service account.
              </>
            }
            placeholder="name@example.com"
            ignore1password
          />
          <InputField
            label="Domain"
            onChange={handleInputChange}
            name="domain"
            value={domain}
            parseTarget
            onBlur={validateForm}
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
            name="privateKey"
            value={privateKey}
            parseTarget
            onBlur={validateForm}
            tooltip={
              <>
                The private key for this Google <br /> Workspace service
                account.
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
