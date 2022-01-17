import React, { useState, useCallback } from "react";
import { connect, useDispatch } from "react-redux";
import { useQuery } from "react-query";
import { size } from "lodash";

// @ts-ignore
import AppConfigForm from "components/forms/admin/AppConfigForm";
import {
  IEnrollSecret,
  IEnrollSecretsResponse,
} from "interfaces/enroll_secret";

import enrollSecretsAPI from "services/entities/enroll_secret";
import configAPI from "services/entities/config";
// @ts-ignore
import { renderFlash } from "redux/nodes/notifications/actions";
// @ts-ignore

export const baseClass = "app-settings";

const AppSettingsPage = (): JSX.Element => {
  const dispatch = useDispatch();

  const {
    data: appConfig,
    isLoading: isLoadingConfig,
    refetch: refetchConfig,
  } = useQuery<any, Error, any>(["config"], () => configAPI.loadAll(), {
    select: (data: any) => data,
  });

  const { data: globalSecrets } = useQuery<
    IEnrollSecretsResponse,
    Error,
    IEnrollSecret[]
  >(["global secrets"], () => enrollSecretsAPI.getGlobalEnrollSecrets(), {
    enabled: true,
    select: (data: IEnrollSecretsResponse) => data.secrets,
  });

  const onFormSubmit = useCallback(
    (formData: any) => {
      console.log("AppSettingsPage formData", formData);
      debugger;
      configAPI
        .update(formData)
        .then(() => {
          dispatch(renderFlash("success", "Successfully updated settings."));
        })
        .catch((errors: any) => {
          if (errors.base) {
            dispatch(renderFlash("error", errors.base));
          }
        })
        .finally(() => {
          refetchConfig();
        });
    },
    [dispatch]
  );

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
              <a href="#organization-info">Organization info</a>
            </li>
            <li>
              <a href="#fleet-web-address">Fleet web address</a>
            </li>
            <li>
              <a href="#saml">SAML single sign on options</a>
            </li>
            <li>
              <a href="#smtp">SMTP options</a>
            </li>
            <li>
              <a href="#osquery-enrollment-secrets">
                Osquery enrollment secrets
              </a>
            </li>
            <li>
              <a href="#agent-options">Global agent options</a>
            </li>
            <li>
              <a href="#host-status-webhook">Host status webhook</a>
            </li>
            <li>
              <a href="#usage-stats">Usage statistics</a>
            </li>
            <li>
              <a href="#advanced-options">Advanced options</a>
            </li>
          </ul>
        </nav>
        {isLoadingConfig ? null : (
          <AppConfigForm
            formData={appConfig}
            handleSubmit={onFormSubmit}
            enrollSecret={globalSecrets}
          />
        )}
      </div>
    </div>
  );
};

export default AppSettingsPage;
