import React, { Component } from "react";
import PropTypes from "prop-types";
import classnames from "classnames";

import AdminDetails from "components/forms/RegistrationForm/AdminDetails";
import ConfirmationPage from "components/forms/RegistrationForm/ConfirmationPage";
import FleetDetails from "components/forms/RegistrationForm/FleetDetails";
import OrgDetails from "components/forms/RegistrationForm/OrgDetails";

const baseClass = "user-registration";

class RegistrationForm extends Component {
  static propTypes = {
    onNextPage: PropTypes.func,
    onSubmit: PropTypes.func,
    page: PropTypes.number,
  };

  constructor(props) {
    super(props);
    const { window } = global;

    this.state = {
      errors: {},
      formData: {
        server_url: window.location.origin,
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
  };

  onSubmitConfirmation = (evt) => {
    evt.preventDefault();

    const { formData } = this.state;
    const { onSubmit: handleSubmit } = this.props;

    return handleSubmit(formData);
  };

  isCurrentPage = (num) => {
    const { page } = this.props;

    if (num === page) {
      return true;
    }

    return false;
  };

  render() {
    const { page } = this.props;
    const { formData } = this.state;
    const { isCurrentPage, onPageFormSubmit, onSubmitConfirmation } = this;

    const adminDetailsContainerClass = classnames(
      `${baseClass}__container`,
      `${baseClass}__container--admin`
    );

    const adminDetailsClass = classnames(
      `${baseClass}__field-wrapper`,
      `${baseClass}__field-wrapper--admin`
    );

    const orgDetailsContainerClass = classnames(
      `${baseClass}__container`,
      `${baseClass}__container--org`
    );

    const orgDetailsClass = classnames(
      `${baseClass}__field-wrapper`,
      `${baseClass}__field-wrapper--org`
    );

    const fleetDetailsContainerClass = classnames(
      `${baseClass}__container`,
      `${baseClass}__container--fleet`
    );

    const fleetDetailsClass = classnames(
      `${baseClass}__field-wrapper`,
      `${baseClass}__field-wrapper--fleet`
    );

    const confirmationContainerClass = classnames(
      `${baseClass}__container`,
      `${baseClass}__container--confirmation`
    );

    const confirmationClass = classnames(
      `${baseClass}__field-wrapper`,
      `${baseClass}__field-wrapper--confirmation`
    );

    const formSectionClasses = classnames(`${baseClass}__form`, {
      [`${baseClass}__form--step1-active`]: page === 1,
      [`${baseClass}__form--step1-complete`]: page > 1,
      [`${baseClass}__form--step2-active`]: page === 2,
      [`${baseClass}__form--step2-complete`]: page > 2,
      [`${baseClass}__form--step3-active`]: page === 3,
      [`${baseClass}__form--step3-complete`]: page > 3,
      [`${baseClass}__form--step4-active`]: page === 4,
    });

    return (
      <div className={baseClass}>
        <div className={formSectionClasses}>
          <div className={adminDetailsContainerClass}>
            <h2>Set up user</h2>
            <AdminDetails
              formData={formData}
              handleSubmit={onPageFormSubmit}
              className={adminDetailsClass}
              currentPage={isCurrentPage(1)}
            />
          </div>
          <div className={orgDetailsContainerClass}>
            <h2>Organization details</h2>
            <OrgDetails
              formData={formData}
              handleSubmit={onPageFormSubmit}
              className={orgDetailsClass}
              currentPage={isCurrentPage(2)}
            />
          </div>
          <div className={fleetDetailsContainerClass}>
            <h2>Set Fleet URL</h2>
            <FleetDetails
              formData={formData}
              handleSubmit={onPageFormSubmit}
              className={fleetDetailsClass}
              currentPage={isCurrentPage(3)}
            />
          </div>
          <div className={confirmationContainerClass}>
            <h2>Confirm configuration</h2>
            <ConfirmationPage
              formData={formData}
              handleSubmit={onSubmitConfirmation}
              className={confirmationClass}
              currentPage={isCurrentPage(4)}
            />
          </div>
        </div>
      </div>
    );
  }
}

export default RegistrationForm;
