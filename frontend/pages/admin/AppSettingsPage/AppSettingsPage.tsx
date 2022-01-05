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
import deepDifference from "utilities/deep_difference";
// @ts-ignore
import { renderFlash } from "redux/nodes/notifications/actions";
// @ts-ignore

export const baseClass = "app-settings";

const AppSettingsPage = (): JSX.Element => {
  const dispatch = useDispatch();

  // ===== local state

  // Unused if we take out configured/enable_smtp replacement in UI
  // const [smtpConfigured, setSmtpConfigured] = useState<any>();
  // const [formData, setFormData] = useState<any>();

  // Old onSubmit function
  // const onFormSubmit = (formData: IFormData) => {
  // const diff = deepDifference(formData, appConfig);

  // dispatch(updateConfig(diff))
  //   .then(() => {
  //     dispatch(renderFlash("success", "Settings updated."));

  //     return false;
  //   })
  //   .catch((errors: any) => {
  //     // TODO: Check out this error handling REP
  //     if (errors.base) {
  //       dispatch(renderFlash("error", errors.base));
  //     }

  //     return false;
  //   });

  //   return false;
  // };

  const onFormSubmit = async (formData: any) => {
    console.log("onFormSubmit formData", formData);
    try {
      // const diff = deepDifference(formData, appConfig);

      // console.log("appConfig", appConfig);
      // console.log("diff", diff);
      // debugger;
      const request = configAPI.update(formData);
      await request.then(() => {
        dispatch(renderFlash("success", "Successfully updated settings."));
        refetchConfig();
      });
    } catch (errors: any) {
      if (errors.base) {
        dispatch(renderFlash("error", errors.base));
      }
    }
  };

  const {
    data: appConfig,
    isLoading: isLoadingConfig,
    refetch: refetchConfig,
  } = useQuery<any, Error, any>(["config"], () => configAPI.loadAll(), {
    select: (data: any) => data,
    // Original configured key replacing the enable_smtp key on frontend
    // onSuccess: (response: any) => {
    // setSmtpConfigured(response.configured);
    // setFormData({ ...response, enable_smtp: smtpConfigured });
    // },
  });

  const {
    isLoading: isGlobalSecretsLoading,
    data: globalSecrets,
    error: loadingGlobalSecretsError,
    refetch: refetchGlobalSecrets,
  } = useQuery<IEnrollSecretsResponse, Error, IEnrollSecret[]>(
    ["global secrets"],
    () => enrollSecretsAPI.getGlobalEnrollSecrets(),
    {
      enabled: true,
      select: (data: IEnrollSecretsResponse) => data.secrets,
    }
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
              <a href="#organization-info">Organization infozzz</a>
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
