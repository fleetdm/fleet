import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';
import { omit, size } from 'lodash';

import AppConfigForm from 'components/forms/admin/AppConfigForm';
import configInterface from 'interfaces/config';
import { createLicense, getLicense } from 'redux/nodes/auth/actions';
import deepDifference from 'utilities/deep_difference';
import licenseInterface from 'interfaces/license';
import { renderFlash } from 'redux/nodes/notifications/actions';
import SmtpWarning from 'components/SmtpWarning';
import { updateConfig } from 'redux/nodes/app/actions';

export const baseClass = 'app-settings';

class AppSettingsPage extends Component {
  static propTypes = {
    appConfig: configInterface,
    dispatch: PropTypes.func.isRequired,
    error: PropTypes.object, // eslint-disable-line react/forbid-prop-types
    license: licenseInterface,
    loadingLicense: PropTypes.bool,
  };

  constructor (props) {
    super(props);

    this.state = { showSmtpWarning: true };
  }

  componentWillMount () {
    const { dispatch } = this.props;

    dispatch(getLicense());

    return false;
  }

  onDismissSmtpWarning = () => {
    this.setState({ showSmtpWarning: false });

    return false;
  }

  onFormSubmit = (formData) => {
    const { appConfig, dispatch } = this.props;
    const appConfigFormData = omit(formData, ['license']);
    const diff = deepDifference(appConfigFormData, appConfig);

    dispatch(updateConfig(diff))
      .then(() => {
        dispatch(renderFlash('success', 'Settings updated!'));

        return false;
      })
      .catch((errors) => {
        if (errors.base) {
          dispatch(renderFlash('error', errors.base));
        }

        return false;
      });

    return false;
  }

  onUpdateLicense = (license) => {
    const { dispatch, license: licenseProp } = this.props;

    if (license === licenseProp.token) {
      return false;
    }

    dispatch(createLicense({ license }))
      .then(() => {
        dispatch(renderFlash('success', 'License updated!'));

        return false;
      })
      .catch(() => false);

    return false;
  }

  render () {
    const { appConfig, error, license, loadingLicense } = this.props;
    const { onDismissSmtpWarning, onFormSubmit, onUpdateLicense } = this;
    const { showSmtpWarning } = this.state;
    const { configured: smtpConfigured } = appConfig;
    const shouldShowWarning = !smtpConfigured && showSmtpWarning;

    if (!size(appConfig) || loadingLicense) {
      return false;
    }

    const formData = { ...appConfig, license: license.token };

    return (
      <div className={`${baseClass} body-wrap`}>
        <h1>App Settings</h1>
        <SmtpWarning
          onDismiss={onDismissSmtpWarning}
          shouldShowWarning={shouldShowWarning}
        />
        <AppConfigForm
          formData={formData}
          handleSubmit={onFormSubmit}
          handleUpdateLicense={onUpdateLicense}
          license={license}
          serverErrors={error}
          smtpConfigured={smtpConfigured}
        />
      </div>
    );
  }
}

const mapStateToProps = ({ app, auth }) => {
  const { config: appConfig, error } = app;
  const { license, loading: loadingLicense } = auth;

  return { appConfig, error, license, loadingLicense };
};

export default connect(mapStateToProps)(AppSettingsPage);
