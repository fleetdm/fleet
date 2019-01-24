import React, { Component } from 'react';
import PropTypes from 'prop-types';
import classnames from 'classnames';

import Button from 'components/buttons/Button';
import formDataInterface from 'interfaces/registration_form_data';
import Icon from 'components/icons/Icon';
import Checkbox from 'components/forms/fields/Checkbox';

const baseClass = 'confirm-user-reg';

class ConfirmationPage extends Component {
  static propTypes = {
    className: PropTypes.string,
    formData: formDataInterface,
    handleSubmit: PropTypes.func,
  };

  importOsqueryConfig = () => {
    const disableImport = true;

    if (disableImport) {
      return false;
    }

    return (
      <div className={`${baseClass}__import`}>
        <Checkbox name="import-install">
          <p>I am migrating an existing <strong>osquery</strong> installation.</p>
          <p>Take me to the <strong>Import Configuration</strong> page.</p>
        </Checkbox>
      </div>
    );
  }

  render () {
    const { importOsqueryConfig } = this;
    const {
      className,
      handleSubmit,
      formData: {
        email,
        kolide_server_url: kolideWebAddress,
        org_name: orgName,
        username,
      },
    } = this.props;

    const confirmRegClasses = classnames(className, baseClass);

    return (
      <form onSubmit={handleSubmit} className={confirmRegClasses}>
        <div className={`${baseClass}__wrapper`}>
          <Icon name="success-check" className={`${baseClass}__icon`} />
          <table className={`${baseClass}__table`}>
            <caption>Administrator Configuration</caption>
            <tbody>
              <tr>
                <th>Username:</th>
                <td>{username}</td>
              </tr>
              <tr>
                <th>Email:</th>
                <td>{email}</td>
              </tr>
              <tr>
                <th>Organization:</th>
                <td>{orgName}</td>
              </tr>
              <tr>
                <th>Fleet URL:</th>
                <td><span className={`${baseClass}__table-url`} title={kolideWebAddress}>{kolideWebAddress}</span></td>
              </tr>
            </tbody>
          </table>

          {importOsqueryConfig()}
        </div>

        <Button type="submit" variant="gradient" className={`${baseClass}__submit`} autofocus>
          Finish
        </Button>
      </form>
    );
  }
}

export default ConfirmationPage;
