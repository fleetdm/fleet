import React, { useCallback, useContext, useState, useEffect } from "react";
import { useQuery } from "react-query";
import { Params } from "react-router/lib/Router";
import { Link } from "react-router";

import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import configAPI from "services/entities/config";
import deepDifference from "utilities/deep_difference";
import { IConfig } from "interfaces/config";
import { IApiError } from "interfaces/errors";
import Spinner from "components/Spinner";
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
  params: Params;
}

export const baseClass = "app-settings";

const AppSettingsPage = ({
  params: { section: sectionTitle },
}: IAppSettingsPageProps): JSX.Element => {
  const { renderFlash } = useContext(NotificationContext);
  const { setConfig } = useContext(AppContext);

  const [activeSection, setActiveSection] = useState<string>("info");

  const { data: appConfig, isLoading, refetch: refetchConfig } = useQuery<
    IConfig,
    Error,
    IConfig
  >(["config"], () => configAPI.loadAll(), {
    select: (data: IConfig) => data,
    onSuccess: (data) => {
      setConfig(data);
    },
  });

  const isNavItemActive = (navItem: string) => {
    return navItem === activeSection ? "active-nav" : "";
  };

  const onFormSubmit = useCallback(
    (formData: IConfig) => {
      if (!appConfig) {
        return false;
      }

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

  useEffect(() => {
    if (sectionTitle) {
      setActiveSection(sectionTitle);
    }
  }, [sectionTitle]);

  const renderSection = () => {
    if (!isLoading && appConfig) {
      return (
        <>
          {activeSection === "info" && (
            <Info appConfig={appConfig} handleSubmit={onFormSubmit} />
          )}
          {activeSection === "webaddress" && (
            <WebAddress appConfig={appConfig} handleSubmit={onFormSubmit} />
          )}
          {activeSection === "sso" && (
            <Sso appConfig={appConfig} handleSubmit={onFormSubmit} />
          )}
          {activeSection === "smtp" && (
            <Smtp appConfig={appConfig} handleSubmit={onFormSubmit} />
          )}
          {activeSection === "agents" && (
            <AgentOptions appConfig={appConfig} handleSubmit={onFormSubmit} />
          )}
          {activeSection === "host-status-webhook" && (
            <HostStatusWebhook
              appConfig={appConfig}
              handleSubmit={onFormSubmit}
            />
          )}
          {activeSection === "statistics" && (
            <Statistics appConfig={appConfig} handleSubmit={onFormSubmit} />
          )}
          {activeSection === "advanced" && (
            <Advanced appConfig={appConfig} handleSubmit={onFormSubmit} />
          )}
        </>
      );
    }

    return <></>;
  };

  return (
    <div className={`${baseClass} body-wrap`}>
      <p className={`${baseClass}__page-description`}>
        Set your organization information and configure SSO and SMTP
      </p>
      {isLoading ? (
        <Spinner />
      ) : (
        <div className={`${baseClass}__settings-form`}>
          <nav>
            <ul className={`${baseClass}__form-nav-list`}>
              <li>
                <Link
                  className={`${baseClass}__nav-link ${isNavItemActive("info")}
                }`}
                  to={PATHS.ADMIN_SETTINGS_INFO}
                >
                  Organization info
                </Link>
              </li>
              <li>
                <Link
                  className={`${baseClass}__nav-link ${isNavItemActive(
                    "webaddress"
                  )}`}
                  to={PATHS.ADMIN_SETTINGS_WEBADDRESS}
                >
                  Fleet web address
                </Link>
              </li>
              <li>
                <Link
                  className={`${baseClass}__nav-link ${isNavItemActive("sso")}`}
                  to={PATHS.ADMIN_SETTINGS_SSO}
                >
                  Single sign-on options
                </Link>
              </li>
              <li>
                <Link
                  className={`${baseClass}__nav-link$ ${isNavItemActive(
                    "smtp"
                  )}`}
                  to={PATHS.ADMIN_SETTINGS_SMTP}
                >
                  SMTP options
                </Link>
              </li>
              <li>
                <Link
                  className={`${baseClass}__nav-link ${isNavItemActive(
                    "agents"
                  )}`}
                  to={PATHS.ADMIN_SETTINGS_AGENTS}
                >
                  Global agent options
                </Link>
              </li>
              <li>
                <Link
                  className={`${baseClass}__nav-link ${isNavItemActive(
                    "host-status-webhook"
                  )}`}
                  to={PATHS.ADMIN_SETTINGS_HOST_STATUS_WEBHOOK}
                >
                  Host status webhook
                </Link>
              </li>
              <li>
                <Link
                  className={`${baseClass}__nav-link ${isNavItemActive(
                    "statistics"
                  )}`}
                  to={PATHS.ADMIN_SETTINGS_STATISTICS}
                >
                  Usage statistics
                </Link>
              </li>
              <li>
                <Link
                  className={`${baseClass}__nav-link ${isNavItemActive(
                    "advanced"
                  )}`}
                  to={PATHS.ADMIN_SETTINGS_ADVANCED}
                >
                  Advanced options
                </Link>
              </li>
            </ul>
          </nav>
          {renderSection()}
        </div>
      )}
    </div>
  );
};

export default AppSettingsPage;
