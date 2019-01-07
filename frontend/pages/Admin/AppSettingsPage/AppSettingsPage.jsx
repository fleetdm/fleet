import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { size } from 'lodash';

import AppConfigForm from 'components/forms/admin/AppConfigForm';
import configInterface from 'interfaces/config';
import deepDifference from 'utilities/deep_difference';
import { renderFlash } from 'redux/nodes/notifications/actions';
import SmtpWarning from 'components/SmtpWarning';
import { updateConfig } from 'redux/nodes/app/actions';

export const baseClass = 'app-settings';

class AppSettingsPage extends Component {
  static propTypes = {
    appConfig: configInterface,
    dispatch: PropTypes.func.isRequired,
    error: PropTypes.object, // eslint-disable-line react/forbid-prop-types
  };

  constructor (props) {
    super(props);

    this.state = { showSmtpWarning: true };
  }

  onDismissSmtpWarning = () => {
    this.setState({ showSmtpWarning: false });

    return false;
  }

  onFormSubmit = (formData) => {
    const { appConfig, dispatch } = this.props;
    const diff = deepDifference(formData, appConfig);

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

  render () {
    const { appConfig, error } = this.props;
    const { onDismissSmtpWarning, onFormSubmit } = this;
    const { showSmtpWarning } = this.state;
    const { configured: smtpConfigured } = appConfig;
    const shouldShowWarning = !smtpConfigured && showSmtpWarning;

    if (!size(appConfig)) {
      return false;
    }

    const formData = { ...appConfig, enable_smtp: smtpConfigured };

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
          serverErrors={error}
          smtpConfigured={smtpConfigured}
        />
      </div>
    );
  }
}

const mapStateToProps = ({ app }) => {
  const { config: appConfig, error } = app;

  return { appConfig, error };
};

export default connect(mapStateToProps)(AppSettingsPage);
