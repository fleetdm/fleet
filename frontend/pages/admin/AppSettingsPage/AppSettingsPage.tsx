import React, { useState } from "react";
import { connect, useDispatch } from "react-redux";
import { size } from "lodash";

// @ts-ignore
import AppConfigForm from "components/forms/admin/AppConfigForm";
import { IConfig } from "interfaces/config";
import { IError } from "interfaces/errors";
import { IEnrollSecret } from "interfaces/enroll_secret";
// @ts-ignore
import deepDifference from "utilities/deep_difference";
// @ts-ignore
import { renderFlash } from "redux/nodes/notifications/actions";
// @ts-ignore
import { updateConfig } from "redux/nodes/app/actions";

export const baseClass = "app-settings";

interface IRootState {
  app: {
    config: IConfig;
    error: IError[];
    enrollSecret: IEnrollSecret;
  };
}

interface IFormData {}

const AppSettingsPage = (): JSX.Element => {
  const dispatch = useDispatch();
  const [showCreateTeamModal, setShowCreateTeamModal] = useState(false);

  const onFormSubmit = (formData: IFormData) => {
    const diff = deepDifference(formData, appConfig);

    dispatch(updateConfig(diff))
      .then(() => {
        dispatch(renderFlash("success", "Settings updated."));

        return false;
      })
      .catch((errors: any) => {
        // TODO: Check out this error handling REP
        if (errors.base) {
          dispatch(renderFlash("error", errors.base));
        }

        return false;
      });

    return false;
  };

  const { configured: smtpConfigured } = appConfig; // there's a interface for config to find these
  const formData = { ...appConfig, enable_smtp: smtpConfigured };

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
        <AppConfigForm
          formData={formData}
          handleSubmit={onFormSubmit}
          serverErrors={error} // this will be handled in the form itself with local state
          smtpConfigured={smtpConfigured}
          enrollSecret={enrollSecret}
        />
      </div>
    </div>
  );
};

export default AppSettingsPage;
