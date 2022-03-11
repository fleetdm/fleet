import React, { useCallback, useContext } from "react";
import { useDispatch } from "react-redux";
import { useQuery } from "react-query";
// @ts-ignore
import { getConfig } from "redux/nodes/app/actions";
// @ts-ignore
import { renderFlash } from "redux/nodes/notifications/actions";

import { AppContext } from "context/app";
import enrollSecretsAPI from "services/entities/enroll_secret";
import configAPI from "services/entities/config";

// @ts-ignore
import deepDifference from "utilities/deep_difference";
import { IConfig, IConfigNested } from "interfaces/config";
import { IApiError } from "interfaces/errors";
import {
  IEnrollSecret,
  IEnrollSecretsResponse,
} from "interfaces/enroll_secret";

// @ts-ignore
import AppConfigForm from "components/forms/admin/AppConfigForm";

export const baseClass = "app-settings";

const AppSettingsPage = (): JSX.Element => {
  const dispatch = useDispatch();

  const { setConfig } = useContext(AppContext);

  const {
    data: appConfig,
    isLoading: isLoadingConfig,
    refetch: refetchConfig,
  } = useQuery<IConfigNested, Error, IConfigNested>(
    ["config"],
    () => configAPI.loadAll(),
    {
      select: (data: IConfigNested) => data,
    }
  );

  const { data: globalSecrets } = useQuery<
    IEnrollSecretsResponse,
    Error,
    IEnrollSecret[]
  >(["global secrets"], () => enrollSecretsAPI.getGlobalEnrollSecrets(), {
    enabled: true,
    select: (data: IEnrollSecretsResponse) => data.secrets,
  });

  const onFormSubmit = useCallback(
    (formData: IConfigNested) => {
      const diff = deepDifference(formData, appConfig);
      // send all formData.agent_options because diff overrides all agent options
      diff.agent_options = formData.agent_options;

      configAPI
        .update(diff)
        .then(() => {
          dispatch(renderFlash("success", "Successfully updated settings."));
        })
        .catch((response: { data: IApiError }) => {
          if (
            response.data.errors[0].reason.includes("could not dial smtp host")
          ) {
            dispatch(
              renderFlash(
                "error",
                "Could not connect to SMTP server. Please try again."
              )
            );
          } else if (response.data.errors) {
            dispatch(
              renderFlash(
                "error",
                `Could not update settings. ${response.data.errors[0].reason}`
              )
            );
          }
        })
        .finally(() => {
          refetchConfig();
          // Config must be updated in both Redux and AppContext
          dispatch(getConfig())
            .then((configState: IConfig) => {
              setConfig(configState);
            })
            .catch(() => false);
        });
    },
    [dispatch, appConfig, getConfig, setConfig]
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

  return (
    <div className={`${baseClass} body-wrap`}>
      <p className={`${baseClass}__page-description`}>
        Set your organization information, Configure SAML and SMTP, and view
        host enroll secrets.
      </p>
      <div className={`${baseClass}__settings-form`}>
        <nav>
          <ul className={`${baseClass}__form-nav-list`}>
            <li>
              <a onClick={() => scrollInto("organization-info")}>
                Organization info
              </a>
            </li>
            <li>
              <a onClick={() => scrollInto("fleet-web-address")}>
                Fleet web address
              </a>
            </li>
            <li>
              <a onClick={() => scrollInto("saml")}>
                SAML single sign on options
              </a>
            </li>
            <li>
              <a onClick={() => scrollInto("smtp")}>SMTP options</a>
            </li>
            <li>
              <a onClick={() => scrollInto("osquery-enrollment-secrets")}>
                Osquery enrollment secrets
              </a>
            </li>
            <li>
              <a onClick={() => scrollInto("agent-options")}>
                Global agent options
              </a>
            </li>
            <li>
              <a onClick={() => scrollInto("host-status-webhook")}>
                Host status webhook
              </a>
            </li>
            <li>
              <a onClick={() => scrollInto("usage-stats")}>Usage statistics</a>
            </li>
            <li>
              <a onClick={() => scrollInto("advanced-options")}>
                Advanced options
              </a>
            </li>
          </ul>
        </nav>
        {!isLoadingConfig && appConfig && (
          <AppConfigForm
            appConfig={appConfig}
            handleSubmit={onFormSubmit}
            enrollSecret={globalSecrets}
          />
        )}
      </div>
    </div>
  );
};

export default AppSettingsPage;
