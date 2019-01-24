import React, { Component } from 'react';
import PropTypes from 'prop-types';
import classnames from 'classnames';

import AdminDetails from 'components/forms/RegistrationForm/AdminDetails';
import ConfirmationPage from 'components/forms/RegistrationForm/ConfirmationPage';
import KolideDetails from 'components/forms/RegistrationForm/KolideDetails';
import OrgDetails from 'components/forms/RegistrationForm/OrgDetails';

const PAGE_HEADER_TEXT = {
  1: 'SET USERNAME & PASSWORD',
  2: 'SET ORGANIZATION DETAILS',
  3: 'SET KOLIDE WEB ADDRESS',
  4: 'SUCCESS',
};

const baseClass = 'user-registration';

class RegistrationForm extends Component {
  static propTypes = {
    onNextPage: PropTypes.func,
    onSubmit: PropTypes.func,
    page: PropTypes.number,
  };

  constructor (props) {
    super(props);
    const { window } = global;

    this.state = {
      errors: {},
      formData: {
        kolide_server_url: window.location.origin,
      },
    };
  }

  onPageFormSubmit = (pageFormData) => {
    const { formData } = this.state;
    const { onNextPage } = this.props;

    this.setState({
      formData: {
        ...formData,
        ...pageFormData,
      },
    });

    return onNextPage();
  }

  onSubmitConfirmation = (evt) => {
    evt.preventDefault();

    const { formData } = this.state;
    const { onSubmit: handleSubmit } = this.props;

    return handleSubmit(formData);
  }

  isCurrentPage = (num) => {
    const { page } = this.props;

    if (num === page) {
      return true;
    }

    return false;
  }

  renderHeader = () => {
    const { page } = this.props;
    const headerText = PAGE_HEADER_TEXT[page];

    if (headerText) {
      return <h2 className={`${baseClass}__title`}>{headerText}</h2>;
    }

    return false;
  }

  renderDescription = () => {
    const { page } = this.props;

    if (page === 1) {
      return (
        <div className={`${baseClass}__description`}>
          <p>Additional admins can be designated within the Fleet App</p>
          <p>Passwords must include 7 characters, at least 1 number (eg. 0-9) and at least 1 symbol (eg. ^&*#)</p>
        </div>
      );
    }

    if (page === 2) {
      return (
        <div className={`${baseClass}__description`}>
          <p>Set your organization&apos;s name (eg. Kolide, Inc)</p>
          <p>(Optional) Set an organization logo to use in the Fleet application. Should be an https URL to an image file (eg. https://kolide.co/logo.png).</p>
        </div>
      );
    }

    if (page === 3) {
      return (
        <div className={`${baseClass}__description`}>
          <p>Define the base URL that clients will use to connect to Fleet.</p>
          <p>
            <small>Note: Please ensure the URL is accessible to all endpoints that will be managed by Fleet. The hostname must match the name on your TLS certificate.</small>
          </p>
        </div>
      );
    }

    return false;
  }

  renderContent = () => {
    const { page } = this.props;
    const { formData } = this.state;
    const {
      onSubmitConfirmation,
      renderDescription,
      renderHeader,
    } = this;

    if (page === 4) {
      return (
        <div>
          {renderHeader()}
          <ConfirmationPage formData={formData} handleSubmit={onSubmitConfirmation} className={`${baseClass}__confirmation`} />
        </div>
      );
    }

    return (
      <div>
        {renderHeader()}
        {renderDescription()}
      </div>
    );
  }

  render () {
    const { page } = this.props;
    const { formData } = this.state;
    const { isCurrentPage, onPageFormSubmit, renderContent } = this;

    const containerClass = classnames(`${baseClass}__container`, {
      [`${baseClass}__container--complete`]: page > 3,
    });

    const adminDetailsClass = classnames(
      `${baseClass}__field-wrapper`,
      `${baseClass}__field-wrapper--admin`
    );

    const orgDetailsClass = classnames(
      `${baseClass}__field-wrapper`,
      `${baseClass}__field-wrapper--org`
    );

    const kolideDetailsClass = classnames(
      `${baseClass}__field-wrapper`,
      `${baseClass}__field-wrapper--kolide`
    );

    const formSectionClasses = classnames(
      `${baseClass}__form`,
      {
        [`${baseClass}__form--step1-active`]: page === 1,
        [`${baseClass}__form--step1-complete`]: page > 1,
        [`${baseClass}__form--step2-active`]: page === 2,
        [`${baseClass}__form--step2-complete`]: page > 2,
        [`${baseClass}__form--step3-active`]: page === 3,
        [`${baseClass}__form--step3-complete`]: page > 3,
      }
    );

    return (
      <div className={baseClass}>
        <div className={containerClass}>
          {renderContent()}

          <div className={formSectionClasses}>
            <AdminDetails formData={formData} handleSubmit={onPageFormSubmit} className={adminDetailsClass} currentPage={isCurrentPage(1)} />

            <OrgDetails formData={formData} handleSubmit={onPageFormSubmit} className={orgDetailsClass} currentPage={isCurrentPage(2)} />

            <KolideDetails formData={formData} handleSubmit={onPageFormSubmit} className={kolideDetailsClass} currentPage={isCurrentPage(3)} />
          </div>
        </div>
      </div>
    );
  }
}

export default RegistrationForm;
