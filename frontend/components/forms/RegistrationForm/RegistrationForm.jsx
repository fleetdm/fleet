import React, { Component, PropTypes } from 'react';

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

class RegistrationForm extends Component {
  static propTypes = {
    onNextPage: PropTypes.func,
    onSubmit: PropTypes.func,
    page: PropTypes.number,
  };

  constructor (props) {
    super(props);

    this.state = { errors: {}, formData: {} };
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

  onSubmit = () => {
    const { formData } = this.state;
    const { onSubmit: handleSubmit } = this.props;

    return handleSubmit(formData);
  }

  renderDescription = () => {
    const { page } = this.props;

    if (page === 1) {
      return (
        <div>
          <p>Additional admins can be designated within the Kolide App</p>
          <p>Passwords must include 7 characters, at least 1 number (eg. 0-9) and at least 1 symbol (eg. ^&*#)</p>
        </div>
      );
    }

    if (page === 2) {
      return (
        <div>
          <p>Set your Organization&apos;s name (eg. Yahoo! Inc)</p>
          <p>Specify the website URL of your organization (eg. Yahoo.com)</p>
        </div>
      );
    }

    if (page === 3) {
      return (
        <div>
          <p>Define the base URL which osqueryd clients use to connect and register with Kolide.</p>
          <p>
            <small>Note: Please ensure the URL you choose is accessible to all endpoints that need to communicate with Kolide. Otherwise, they will not be able to correctly register.</small>
          </p>
        </div>
      );
    }

    return false;
  }

  renderHeader = () => {
    const { page } = this.props;
    const headerText = PAGE_HEADER_TEXT[page];

    if (headerText) {
      return <h2>{headerText}</h2>;
    }

    return false;
  }

  renderPageForm = () => {
    const { formData } = this.state;
    const { onPageFormSubmit, onSubmit } = this;
    const { page } = this.props;

    if (page === 1) {
      return <AdminDetails formData={formData} handleSubmit={onPageFormSubmit} />;
    }

    if (page === 2) {
      return <OrgDetails formData={formData} handleSubmit={onPageFormSubmit} />;
    }

    if (page === 3) {
      return <KolideDetails formData={formData} handleSubmit={onPageFormSubmit} />;
    }

    if (page === 4) {
      return <ConfirmationPage formData={formData} handleSubmit={onSubmit} />;
    }

    return false;
  }

  render () {
    const { onSubmit } = this.props;
    const { renderDescription, renderHeader, renderPageForm } = this;

    return (
      <form onSubmit={onSubmit}>
        {renderHeader()}
        {renderDescription()}
        {renderPageForm()}
      </form>
    );
  }
}

export default RegistrationForm;
