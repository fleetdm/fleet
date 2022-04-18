import React, { useCallback, useContext, useState } from "react";
import { useQuery } from "react-query";
import { AppContext } from "context/app";
import { NotificationContext } from "context/notification"; // @ts-ignore

import configAPI from "services/entities/config";

// @ts-ignore
import deepDifference from "utilities/deep_difference";
import { IConfig } from "interfaces/config";
import { IApiError } from "interfaces/errors";

import Button from "components/buttons/Button";
import OrganizationInfo from "./cards/OrganizationInfo";
import FleetWebAddress from "./cards/FleetWebAddress";
import Saml from "./cards/Saml";
import Smtp from "./cards/Smtp";
import AgentOptions from "./cards/AgentOptions";
import HostStatusWebhook from "./cards/HostStatusWebhook";
import UsageStats from "./cards/UsageStats";
import AdvancedOptions from "./cards/AdvancedOptions";

// import AppConfigForm from "components/forms/admin/AppConfigForm";

export const baseClass = "app-settings";

const AppSettingsPage = (): JSX.Element => {
  const { renderFlash } = useContext(NotificationContext);
  const { setConfig } = useContext(AppContext);

  const [showOrgInfo, setShowOrgInfo] = useState<boolean>(true);
  const [showFleetWebAddress, setShowFleetWebAddress] = useState<boolean>(
    false
  );
  const [showSaml, setShowSaml] = useState<boolean>(false);
  const [showSmtp, setShowSmtp] = useState<boolean>(false);
  const [showAgentOptions, setShowAgentOptions] = useState<boolean>(false);
  const [showHostStatusWebhook, setShowHostStatusWebhook] = useState<boolean>(
    false
  );
  const [showUsageStats, setShowUsageStats] = useState<boolean>(false);
  const [showAdvancedOptions, setShowAdvancedOptions] = useState<boolean>(
    false
  );

  const {
    data: appConfig,
    isLoading: isLoadingConfig,
    refetch: refetchConfig,
  } = useQuery<IConfig, Error, IConfig>(["config"], () => configAPI.loadAll(), {
    select: (data: IConfig) => data,
    onSuccess: (data) => {
      setConfig(data);
    },
  });

  const onFormSubmit = useCallback(
    (formData: IConfig) => {
      const diff = deepDifference(formData, appConfig);
      // send all formData.agent_options because diff overrides all agent options
      diff.agent_options = formData.agent_options;

      configAPI
        .update(diff)
        .then(() => {
          renderFlash("success", "Successfully updated settings.");
        })
        .catch((response: { data: IApiError }) => {
          if (
            response?.data.errors[0].reason.includes("could not dial smtp host")
          ) {
            renderFlash(
              "error",
              "Could not connect to SMTP server. Please try again."
            );
          } else if (response?.data.errors) {
            renderFlash(
              "error",
              `Could not update settings. ${response.data.errors[0].reason}`
            );
          }
        })
        .finally(() => {
          refetchConfig();
        });
    },
    [appConfig]
  );

  // WHY???
  // Because Firefox and Safari don't support anchor links :-(
  const scrollInto = (elementId: string) => {
    const yOffset = -215; // headers and tabs
    const element = document.getElementById(elementId);

    if (element) {
      const top =
        element.getBoundingClientRect().top + window.pageYOffset + yOffset;
      window.scrollTo({ top });
    }
  };

  const showSection = (linkString: string) => {
    setShowOrgInfo(false);
    setShowFleetWebAddress(false);
    setShowSaml(false);
    setShowSmtp(false);
    setShowAgentOptions(false);
    setShowHostStatusWebhook(false);
    setShowUsageStats(false);
    setShowAdvancedOptions(false);

    switch (linkString) {
      case "org-info":
        setShowOrgInfo(true);
        return;
      case "fleet-web-address":
        setShowFleetWebAddress(true);
        return;
      case "saml":
        setShowSaml(true);
        return;
      case "smtp":
        setShowSmtp(true);
        return;
      case "agent-options":
        setShowAgentOptions(true);
        return;
      case "host-status-webhook":
        setShowHostStatusWebhook(true);
        return;
      case "usage-stats":
        setShowUsageStats(true);
        return;
      default:
        setShowAdvancedOptions(true);
        return;
    }
  };

  const renderSection = () => {
    if (!isLoadingConfig && appConfig) {
      return (
        <>
          {showOrgInfo && (
            <OrganizationInfo
              appConfig={appConfig}
              handleSubmit={onFormSubmit}
            />
          )}
          {showFleetWebAddress && (
            <FleetWebAddress
              appConfig={appConfig}
              handleSubmit={onFormSubmit}
            />
          )}
          {showSaml && (
            <Saml appConfig={appConfig} handleSubmit={onFormSubmit} />
          )}
          {showSmtp && (
            <Smtp appConfig={appConfig} handleSubmit={onFormSubmit} />
          )}
          {showAgentOptions && (
            <AgentOptions appConfig={appConfig} handleSubmit={onFormSubmit} />
          )}
          {showHostStatusWebhook && (
            <HostStatusWebhook
              appConfig={appConfig}
              handleSubmit={onFormSubmit}
            />
          )}
          {showUsageStats && (
            <UsageStats appConfig={appConfig} handleSubmit={onFormSubmit} />
          )}
          {showAdvancedOptions && (
            <AdvancedOptions
              appConfig={appConfig}
              handleSubmit={onFormSubmit}
            />
          )}
          {/* <AppConfigForm appConfig={appConfig} handleSubmit={onFormSubmit} /> */}
        </>
      );
    }
  };
  return (
    <div className={`${baseClass} body-wrap`}>
      <p className={`${baseClass}__page-description`}>
        Set your organization information and configure SAML and SMTP.
      </p>
      <div className={`${baseClass}__settings-form`}>
        <nav>
          <ul className={`${baseClass}__form-nav-list`}>
            <li>
              <Button
                className={`${baseClass}__nav-button`}
                variant="text-nav"
                onClick={() => showSection("org-info")}
              >
                Organization info
              </Button>
            </li>
            <li>
              <Button
                className={`${baseClass}__nav-button`}
                variant="text-nav"
                onClick={() => showSection("fleet-web-address")}
              >
                Fleet web address
              </Button>
            </li>
            <li>
              <Button
                className={`${baseClass}__nav-button`}
                variant="text-nav"
                onClick={() => showSection("saml")}
              >
                SAML single sign on options
              </Button>
            </li>
            <li>
              <Button
                className={`${baseClass}__nav-button`}
                variant="text-nav"
                onClick={() => showSection("smtp")}
              >
                SMTP options
              </Button>
            </li>
            <li>
              <Button
                className={`${baseClass}__nav-button`}
                variant="text-nav"
                onClick={() => showSection("agent-options")}
              >
                Global agent options
              </Button>
            </li>
            <li>
              <Button
                className={`${baseClass}__nav-button`}
                variant="text-nav"
                onClick={() => showSection("host-status-webhook")}
              >
                Host status webhook
              </Button>
            </li>
            <li>
              <Button
                className={`${baseClass}__nav-button`}
                variant="text-nav"
                onClick={() => showSection("usage-stats")}
              >
                Usage statistics
              </Button>
            </li>
            <li>
              <Button
                className={`${baseClass}__nav-button`}
                variant="text-nav"
                onClick={() => showSection("advanced-options")}
              >
                Advanced options
              </Button>
            </li>
          </ul>
        </nav>
        {renderSection()}
      </div>
    </div>
  );
};

export default AppSettingsPage;
