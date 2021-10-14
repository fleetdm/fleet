import React, { Component } from "react";
import PropTypes from "prop-types";
import { connect } from "react-redux";
import { size } from "lodash";

import AppConfigForm from "components/forms/admin/AppConfigForm";
import configInterface from "interfaces/config";
import enrollSecretInterface from "interfaces/enroll_secret";
import deepDifference from "utilities/deep_difference";
import { renderFlash } from "redux/nodes/notifications/actions";
import { updateConfig } from "redux/nodes/app/actions";

export const baseClass = "app-settings";
class AppSettingsPage extends Component {
  static propTypes = {
    appConfig: configInterface,
    dispatch: PropTypes.func.isRequired,
    error: PropTypes.object, // eslint-disable-line react/forbid-prop-types
    enrollSecret: PropTypes.arrayOf(enrollSecretInterface),
  };

  onFormSubmit = (formData) => {
    const { appConfig, dispatch } = this.props;
    const diff = deepDifference(formData, appConfig);

    dispatch(updateConfig(diff))
      .then(() => {
        dispatch(renderFlash("success", "Settings updated."));

        return false;
      })
      .catch((errors) => {
        if (errors.base) {
          dispatch(renderFlash("error", errors.base));
        }

        return false;
      });

    return false;
  };

  render() {
    const { appConfig, error, enrollSecret } = this.props;
    const { onFormSubmit } = this;
    const { configured: smtpConfigured } = appConfig;

    if (!size(appConfig)) {
      return false;
    }

    const formData = { ...appConfig, enable_smtp: smtpConfigured };

    const scrollTo = (elementId) => {
      document.getElementById(elementId).scrollIntoView(true);
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
                <a onClick={() => scrollTo("organization-info")}>
                  Organization info
                </a>
              </li>
              <li>
                <a onClick={() => scrollTo("fleet-web-address")}>
                  Fleet web address
                </a>
              </li>
              <li>
                <a onClick={() => scrollTo("saml")}>
                  SAML single sign on options
                </a>
              </li>
              <li>
                <a onClick={() => scrollTo("smtp")}>SMTP options</a>
              </li>
              <li>
                <a onClick={() => scrollTo("osquery-enrollment-secrets")}>
                  Osquery enrollment secrets
                </a>
              </li>
              <li>
                <a onClick={() => scrollTo("agent-options")}>
                  Global agent options
                </a>
              </li>
              <li>
                <a onClick={() => scrollTo("host-status-webhook")}>
                  Host status webhook
                </a>
              </li>
              <li>
                <a onClick={() => scrollTo("usage-stats")}>Usage statistics</a>
              </li>
              <li>
                <a onClick={() => scrollTo("advanced-options")}>
                  Advanced options
                </a>
              </li>
            </ul>
          </nav>
          <AppConfigForm
            formData={formData}
            handleSubmit={onFormSubmit}
            serverErrors={error}
            smtpConfigured={smtpConfigured}
            enrollSecret={enrollSecret}
          />
        </div>
      </div>
    );
  }
}

const mapStateToProps = ({ app }) => {
  const { config: appConfig, error, enrollSecret } = app;

  return { appConfig, error, enrollSecret };
};

export default connect(mapStateToProps)(AppSettingsPage);
