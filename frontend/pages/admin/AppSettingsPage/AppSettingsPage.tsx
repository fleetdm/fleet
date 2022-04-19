import React, { useCallback, useContext, useState, useEffect } from "react";
import { useQuery } from "react-query";
import { InjectedRouter, Params } from "react-router/lib/Router";
import { Link } from "react-router";

import { AppContext } from "context/app";
import { NotificationContext } from "context/notification"; // @ts-ignore

import configAPI from "services/entities/config";

// @ts-ignore
import deepDifference from "utilities/deep_difference";
import { IConfig } from "interfaces/config";
import { IApiError } from "interfaces/errors";

import PATHS from "router/paths";
import Info from "./cards/Info";
import WebAddress from "./cards/WebAddress";
import Sso from "./cards/Sso";
import Smtp from "./cards/Smtp";
import AgentOptions from "./cards/Agents";
import HostStatusWebhook from "./cards/HostStatusWebhook";
import Statistics from "./cards/Statistics";
import Advanced from "./cards/Advanced";

interface IAppSettingsPageProps {
  router: InjectedRouter; // v3
  params: Params;
}

export const baseClass = "app-settings";

const AppSettingsPage = ({
  router,
  params: { section: sectionTitle },
}: IAppSettingsPageProps): JSX.Element => {
  const { renderFlash } = useContext(NotificationContext);
  const { setConfig } = useContext(AppContext);

  const [showOrgInfo, setShowOrgInfo] = useState<boolean>(
    sectionTitle === "info"
  );
  const [showFleetWebAddress, setShowFleetWebAddress] = useState<boolean>(
    sectionTitle === "webaddress"
  );
  const [showSso, setShowSso] = useState<boolean>(sectionTitle === "sso");
  const [showSmtp, setShowSmtp] = useState<boolean>(sectionTitle === "smtp");
  const [showAgentOptions, setShowAgentOptions] = useState<boolean>(
    sectionTitle === "agents"
  );
  const [showHostStatusWebhook, setShowHostStatusWebhook] = useState<boolean>(
    sectionTitle === "host-status-webhook"
  );
  const [showUsageStats, setShowUsageStats] = useState<boolean>(
    sectionTitle === "statistics"
  );
  const [showAdvancedOptions, setShowAdvancedOptions] = useState<boolean>(
    sectionTitle === "advanced"
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

  const showSection = (linkString: string) => {
    setShowOrgInfo(false);
    setShowFleetWebAddress(false);
    setShowSso(false);
    setShowSmtp(false);
    setShowAgentOptions(false);
    setShowHostStatusWebhook(false);
    setShowUsageStats(false);
    setShowAdvancedOptions(false);

    switch (linkString) {
      case "webaddress":
        setShowFleetWebAddress(true);
        return;
      case "sso":
        setShowSso(true);
        return;
      case "smtp":
        setShowSmtp(true);
        return;
      case "agents":
        setShowAgentOptions(true);
        return;
      case "host-status-webhook":
        setShowHostStatusWebhook(true);
        return;
      case "statistics":
        setShowUsageStats(true);
        return;
      case "advanced":
        setShowAdvancedOptions(true);
        return;
      default:
        setShowOrgInfo(true);
        return;
    }
  };

  useEffect(() => {
    showSection(sectionTitle);
  }, [sectionTitle]);

  const renderSection = () => {
    if (!isLoadingConfig && appConfig) {
      return (
        <>
          {showOrgInfo && (
            <Info appConfig={appConfig} handleSubmit={onFormSubmit} />
          )}
          {showFleetWebAddress && (
            <WebAddress appConfig={appConfig} handleSubmit={onFormSubmit} />
          )}
          {showSso && <Sso appConfig={appConfig} handleSubmit={onFormSubmit} />}
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
            <Statistics appConfig={appConfig} handleSubmit={onFormSubmit} />
          )}
          {showAdvancedOptions && (
            <Advanced appConfig={appConfig} handleSubmit={onFormSubmit} />
          )}
        </>
      );
    }
  };
  return (
    <div className={`${baseClass} body-wrap`}>
      <p className={`${baseClass}__page-description`}>
        Set your organization information and configure SSO and SMTP
      </p>
      <div className={`${baseClass}__settings-form`}>
        <nav>
          <ul className={`${baseClass}__form-nav-list`}>
            <li>
              <Link
                className={`${baseClass}__nav-link`}
                to={PATHS.ADMIN_SETTINGS_INFO}
              >
                Organization info
              </Link>
            </li>
            <li>
              <Link
                className={`${baseClass}__nav-link`}
                to={PATHS.ADMIN_SETTINGS_WEBADDRESS}
              >
                Fleet web address
              </Link>
            </li>
            <li>
              <Link
                className={`${baseClass}__nav-link`}
                to={PATHS.ADMIN_SETTINGS_SSO}
              >
                Single sign-on options
              </Link>
            </li>
            <li>
              <Link
                className={`${baseClass}__nav-link`}
                to={PATHS.ADMIN_SETTINGS_SMTP}
              >
                SMTP options
              </Link>
            </li>
            <li>
              <Link
                className={`${baseClass}__nav-link`}
                to={PATHS.ADMIN_SETTINGS_AGENTS}
              >
                Global agent options
              </Link>
            </li>
            <li>
              <Link
                className={`${baseClass}__nav-link`}
                to={PATHS.ADMIN_SETTINGS_HOST_STATUS_WEBHOOK}
              >
                Host status webhook
              </Link>
            </li>
            <li>
              <Link
                className={`${baseClass}__nav-link`}
                to={PATHS.ADMIN_SETTINGS_STATISTICS}
              >
                Usage statistics
              </Link>
            </li>
            <li>
              <Link
                className={`${baseClass}__nav-link`}
                to={PATHS.ADMIN_SETTINGS_ADVANCED}
              >
                Advanced options
              </Link>
            </li>
          </ul>
        </nav>
        {renderSection()}
      </div>
    </div>
  );
};

export default AppSettingsPage;
