import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';
import { size } from 'lodash';

import AppConfigForm from 'components/forms/admin/AppConfigForm';
import configInterface from 'interfaces/config';
import { renderFlash } from 'redux/nodes/notifications/actions';
import SmtpWarning from 'pages/Admin/AppSettingsPage/SmtpWarning';
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
    const { dispatch } = this.props;

    dispatch(updateConfig(formData))
      .then(() => {
        dispatch(renderFlash('success', 'Settings updated!'));
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

    return (
      <div className={`${baseClass} body-wrap`}>
        <h1>App Settings</h1>
        <SmtpWarning
          onDismiss={onDismissSmtpWarning}
          shouldShowWarning={shouldShowWarning}
        />
        <AppConfigForm
          formData={appConfig}
          errors={error}
          handleSubmit={onFormSubmit}
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
